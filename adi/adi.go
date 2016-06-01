package adi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

type Response struct {
	Text   string
	Charge bool
}

type Level uint8

type Points uint64

type User struct {
	ID     string `json:"id"`
	Level  Level  `json:"level"`
	Points Points `json:"points"`
}

type Bank struct {
	Points  Points `json:"points"`
	Lottery struct {
		Pot         Points            `json:"pot"`
		LastDraw    time.Time         `json:"last_draw"`
		DrawEvery   time.Duration     `json:"draw_every"`
		TicketsSold uint64            `json:"tickets_sold"`
		Tickets     map[string]uint64 `json:"tickets"`
		Invest      Points            `json:"invest"`
		TicketPrice Points            `json:"ticket_price"`
	} `json:"lottery"`
}

type Account interface {
	Add(Points)
	Sub(Points)
	Balance() Points
}

type CommandFunc func(text string, u *User, rtm *slack.RTM) Response

type Command struct {
	Name          string      `json:"name"`
	RequiredLevel Level       `json:"required_level"`
	Price         Points      `json:"price"`
	Visible       bool        `json:"visible"`
	Proxy         string      `json:"proxy,omitempty"`
	Func          CommandFunc `json:"-"`
}

var (
	DubtrackRoom string
	DefaultLevel Level
	GlobalBank   Bank
	Users        []User
	Commands     []Command

	helpString   string
	commandFuncs = map[string]CommandFunc{}
	reCommand    *regexp.Regexp
	reUrlUnFurl  = regexp.MustCompile(
		"<((?:https?|ftp)://[^|>]+)(?:|[^>]+)?>")
)

func (u *Points) Balance() Points { return *u }

func (u *Points) Add(p Points) {
	if *u > (Points(math.MaxUint64) - p) {
		*u = Points(math.MaxUint64)
	} else {
		*u += Points(p)
	}
}

func (u *Points) Sub(p Points) {
	if *u < p {
		*u = 0
	} else {
		*u -= p
	}
}

func GetCommandByName(name string) *Command {
	for i, _ := range Commands {
		if Commands[i].Name == name {
			return &Commands[i]
		}
	}
	return nil
}

func RandBool() bool {
	var bn big.Int
	bn.SetUint64(2)
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return false
	}
	return r.Uint64() == 1
}

func RandUint64(n uint64) uint64 {
	var bn big.Int
	bn.SetUint64(n)
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return 0
	}
	return r.Uint64()
}

func RandUint32(n uint32) uint32 {
	var bn big.Int
	bn.SetUint64(uint64(n))
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return 0
	}
	return uint32(r.Uint64())
}

func MulOverflows(a, b uint64) bool {
	if a <= 1 || b <= 1 {
		return false
	}
	c := a * b
	return c/b != a
}

func getGeneralChat(rtm *slack.RTM) *slack.Channel {
	cs, err := rtm.GetChannels(true)
	if err != nil {
		log.Fatal("ERROR:", err)
		return nil
	}
	for _, c := range cs {
		if c.IsGeneral {
			return &c
		}
	}
	return nil
}

func GetUserByName(rtm *slack.RTM, name string) *slack.User {
	us, err := rtm.GetUsers()
	if err != nil {
		log.Println("ERROR:", err.Error())
		return nil
	}
	for _, o := range us {
		if o.Name == name {
			return &o
		}
	}
	return nil
}

func GetAccountByName(rtm *slack.RTM, name string) Account {
	if name == "bank" {
		return &GlobalBank.Points
	}
	if name == "pot" {
		return &GlobalBank.Lottery.Pot
	}
	us := GetUserByName(rtm, name)
	if us == nil {
		return nil
	}
	return &GetCreateUser(us.ID).Points
}

func ParsePoints(src Account, name, text string) (Points, string) {
	var n Points
	if text == "all" {
		if src.Balance() == 0 {
			if len(name) == 0 {
				return 0, "you have no points"
			} else {
				return 0, fmt.Sprintf("%s got no points", name)
			}
		}
		n = src.Balance()
	} else {
		t, err := strconv.ParseUint(text, 10, 64)
		if err != nil || t == 0 {
			return 0, "points have to be positive"
		}
		if src.Balance() < Points(t) {
			if len(name) == 0 {
				return 0, fmt.Sprintf(
					"you do not have enough points. you have %d",
					src.Balance())
			} else {
				return 0, fmt.Sprintf(
					"%s does not have enough points. %s has %d",
					name, name, src.Balance())
			}
		}
		n = Points(t)
	}
	return n, ""
}

func GetCreateUser(id string) *User {
	for i, _ := range Users {
		if Users[i].ID == id {
			return &Users[i]
		}
	}
	user := User{ID: id, Level: DefaultLevel, Points: 0}
	Users = append(Users, user)
	return &Users[len(Users)-1]
}

func drawLottery(rtm *slack.RTM) {
	lot := &GlobalBank.Lottery
	nextDraw := lot.LastDraw.Add(GlobalBank.Lottery.DrawEvery).UTC()
	if time.Now().UTC().Before(nextDraw) {
		return
	}
	if lot.TicketsSold == 0 {
		lot.LastDraw = time.Now().UTC()
		return
	}
	if len(lot.Tickets) >= 2 {
		var o uint64
		var i uint32
		participants := make([]struct {
			ID        string
			Low, High uint64
		}, len(lot.Tickets))
		for p, n := range lot.Tickets {
			participants[i].ID = p
			participants[i].Low = o
			participants[i].High = o + n - 1
			o += n
			i++
		}
		r := RandUint64(o)
		var w string
		for _, p := range participants {
			if r >= p.Low && r <= p.High {
				w = p.ID
				break
			}
		}
		var dst Account
		dst = &GetCreateUser(w).Points
		var name string
		{
			us, err := rtm.GetUserInfo(w)
			if err != nil {
				log.Println("ERROR:", err)
				name = "somebody"
			} else {
				name = us.Name
			}
		}
		m := fmt.Sprintf("%s won the lottery pot of %d points",
			name,
			lot.Pot,
		)
		log.Println(m)
		cg := getGeneralChat(rtm)
		if cg != nil {
			rtm.SendMessage(rtm.NewOutgoingMessage(m, cg.ID))
		}
		dst.Add(lot.Pot)
		lot.Pot = 0
		lot.TicketsSold = 0
		lot.Tickets = map[string]uint64{}
	}
	bi := lot.Invest
	if GlobalBank.Points.Balance() > bi {
		GlobalBank.Points.Sub(bi)
		lot.Pot.Add(bi)
	}
	lot.LastDraw = time.Now().UTC()
}

func ResetCommands() {
	commandStrings := make([]string, len(Commands))
	for i, _ := range Commands {
		commandStrings[i] = Commands[i].Name
	}
	reCommand = regexp.MustCompile(
		fmt.Sprintf("(?s)^(%s)(?:\\s+(.+))?\\s*$",
			strings.Join(commandStrings, "|")))
}

func parseCommand(text string) (*Command, string, error) {
	m := reCommand.FindStringSubmatch(text)
	if m == nil {
		return nil, "", errors.New("commands: " + helpString)
	}
	cmd := GetCommandByName(m[1])
	if cmd == nil {
		return nil, "", errors.New("commands: " + helpString)
	}
	return cmd, m[2], nil
}

func UrlUnFurl(furl string) string {
	b := []byte(furl)
	for {
		m := reUrlUnFurl.FindSubmatchIndex(b)
		if m == nil {
			break
		}
		d := (m[1] - m[0]) - (m[3] - m[2])
		l := len(b) - d
		copy(b[m[0]:], b[m[2]:m[3]])
		copy(b[m[3]-1:l], b[m[1]:])
		b = b[:l]
	}
	return string(b)
}

func RegisterFunc(name string, f CommandFunc) {
	commandFuncs[name] = f
}

func Run() {
	{
		f, err := os.OpenFile("./log",
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0750)
		if err != nil {
			log.Panicln(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	var (
		key              string
		shortCommands    bool
		shortCommandSign string
	)
	{
		var config struct {
			Key              string `json:"key"`
			ShortCommands    bool   `json:"short_commands"`
			ShortCommandSign string `json:"short_command_sign"`
			DefaultLevel     Level  `json:"default_level"`
			DubtrackRoom     string `json:"dubtrack_room"`
		}
		fd, err := os.OpenFile("./config.json", os.O_RDONLY, 0750)
		if err != nil {
			log.Panicln(err)
		}
		if err := json.NewDecoder(fd).Decode(&config); err != nil {
			fd.Close()
			log.Panicln(err)
		}
		fd.Close()
		key = config.Key
		shortCommands = config.ShortCommands
		shortCommandSign = config.ShortCommandSign
		DefaultLevel = config.DefaultLevel
		DubtrackRoom = config.DubtrackRoom
	}
	{
		readDump := func(file string, item interface{}) {
			fd, err := os.OpenFile(file, os.O_RDONLY, 0750)
			if err != nil {
				log.Panicln(err)
			}
			if err := json.NewDecoder(fd).Decode(item); err != nil {
				fd.Close()
				log.Panicln(err)
			}
			fd.Close()
		}
		Commands = make([]Command, 0, 10)
		readDump("./commands.json", &Commands)
		Users = make([]User, 0, 10)
		readDump("./users.json", &Users)
		readDump("./bank.json", &GlobalBank)
	}
	{
		visibleCmds := make([]string, 0, len(Commands))
		commandStrings := make([]string, len(Commands))
		for i, _ := range Commands {
			name := Commands[i].Name
			f, ok := commandFuncs[name]
			if ok {
				Commands[i].Func = f
			}
			commandStrings[i] = name
			if Commands[i].Visible {
				visibleCmds = append(visibleCmds, name)
			}
		}
		sort.Sort(sort.StringSlice(visibleCmds))
		helpString = strings.Join(visibleCmds, ", ")
		reCommand = regexp.MustCompile(
			fmt.Sprintf("(?s)^(%s)(?:\\s+(.+))?\\s*$",
				strings.Join(commandStrings, "|")))
	}
	var reToMe *regexp.Regexp
	tick := time.NewTicker(time.Minute)
	api := slack.New(key)
	api.SetDebug(false)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectedEvent:
				if shortCommands {
					reToMe = regexp.MustCompile(fmt.Sprintf("^(?:<@%s>\\s*|%s)",
						rtm.GetInfo().User.ID,
						shortCommandSign))
				} else {
					reToMe = regexp.MustCompile(fmt.Sprintf("^<@%s>\\s*",
						rtm.GetInfo().User.ID))
				}
			case *slack.MessageEvent:
				{
					m := reToMe.FindStringSubmatch(ev.Text)
					if m == nil {
						continue Loop
					}
					ev.Text = ev.Text[len(m[0]):]
				}
				log.Println(ev.User, ev.Text)
				cmd, params, err := parseCommand(ev.Text)
				if err != nil {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						err.Error(), ev.Channel))
					continue Loop
				}
				u := GetCreateUser(ev.User)
				if u.Level < cmd.RequiredLevel {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf(
							"unprivileged. your level: %d. required: %d",
							u.Level, cmd.RequiredLevel), ev.Channel))
					continue Loop
				}
				if cmd.Price > u.Points {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf(
							"not enough points. your points: %d. required: %d",
							u.Points, cmd.Price), ev.Channel))
					continue Loop
				}
				var r Response
				if cmd.Proxy != "" {
					var nc string
					if strings.Contains(cmd.Proxy, "%s") {
						nc = fmt.Sprintf(cmd.Proxy, params)
					} else {
						nc = cmd.Proxy
					}
					cmd, params, err := parseCommand(nc)
					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							err.Error(), ev.Channel))
						continue Loop
					}
					r = cmd.Func(params, u, rtm)
				} else {
					r = cmd.Func(params, u, rtm)
				}
				if r.Text != "" {
					rtm.SendMessage(rtm.NewOutgoingMessage(r.Text, ev.Channel))
				}
				if cmd.Price > 0 && r.Charge {
					u.Points.Sub(cmd.Price)
					GlobalBank.Points.Add(cmd.Price)
				}
			case *slack.PresenceChangeEvent:
			case *slack.LatencyReport:
			case *slack.RTMError:
				log.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				log.Println("invalid credentials")
				break Loop
			default:
			}
		case <-tick.C:
			{
				drawLottery(rtm)
				{ // salary
					us, _ := rtm.GetUsers()
					for _, o := range us {
						if GlobalBank.Points == 0 ||
							o.IsBot ||
							o.ID == "USLACKBOT" ||
							o.Presence != "active" {
							continue
						}
						uo := GetCreateUser(o.ID)
						GlobalBank.Points.Sub(1)
						uo.Points.Add(1)
					}
				}
				writeDump := func(file string, item interface{}) {
					fd, err := os.OpenFile(file,
						os.O_WRONLY|os.O_TRUNC, 0750)
					if err != nil {
						log.Println(err)
						return
					}
					if err := json.NewEncoder(fd).Encode(item); err != nil {
						fd.Close()
						log.Println(err)
						return
					}
					fd.Close()
				}
				writeDump("./users.json", Users)
				writeDump("./commands.json", Commands)
				writeDump("./bank.json", GlobalBank)
			}
		}
	}
}
