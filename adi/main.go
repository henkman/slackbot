package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/henkman/google"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
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
		BankInvest  Points            `json:"bank_invest"`
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
	Func          CommandFunc `json:"-"`
}

const (
	TLD = "de"
)

var (
	defaultLevel Level
	cvm          *otto.Otto
	gclient      google.Client
	bank         Bank
	users        []User
	commands     []Command
	commandFuncs = map[string]CommandFunc{}
	helpString   string
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

func getCommandByName(name string) *Command {
	for i, _ := range commands {
		if commands[i].Name == name {
			return &commands[i]
		}
	}
	return nil
}

func randBool() bool {
	var bn big.Int
	bn.SetUint64(2)
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return false
	}
	return r.Uint64() == 1
}

func randUint64(n uint64) uint64 {
	var bn big.Int
	bn.SetUint64(n)
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return 0
	}
	return r.Uint64()
}

func randUint32(n uint32) uint32 {
	var bn big.Int
	bn.SetUint64(uint64(n))
	r, err := rand.Int(rand.Reader, &bn)
	if err != nil {
		log.Println(err)
		return 0
	}
	return uint32(r.Uint64())
}

func mulOverflows(a, b uint64) bool {
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

func drawLottery(rtm *slack.RTM) {
	lot := &bank.Lottery
	nextDraw := lot.LastDraw.Add(bank.Lottery.DrawEvery).UTC()
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
		r := randUint64(o)
		var w string
		for _, p := range participants {
			if r >= p.Low && r <= p.High {
				w = p.ID
				break
			}
		}
		var dst Account
		dst = &getCreateUser(w).Points
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
	bi := lot.BankInvest
	if bank.Points.Balance() > bi {
		bank.Points.Sub(bi)
		lot.Pot.Add(bi)
	}
	lot.LastDraw = time.Now().UTC()
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
		defaultLevel = config.DefaultLevel
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
		commands = make([]Command, 0, 10)
		readDump("./commands.json", &commands)
		users = make([]User, 0, 10)
		readDump("./users.json", &users)
		readDump("./bank.json", &bank)
	}
	var reCommand *regexp.Regexp
	{
		visibleCmds := make([]string, 0, len(commands))
		commandStrings := make([]string, len(commands))
		for i, _ := range commands {
			name := commands[i].Name
			f, ok := commandFuncs[name]
			if ok {
				commands[i].Func = f
			} else {
				commands[i].Func = func(text string, u *User, rtm *slack.RTM) Response {
					return Response{
						Text: "not implemented",
					}
				}
			}
			commandStrings[i] = name
			if commands[i].Visible {
				visibleCmds = append(visibleCmds, name)
			}
		}
		sort.Sort(sort.StringSlice(visibleCmds))
		helpString = strings.Join(visibleCmds, ", ")
		reCommand = regexp.MustCompile(
			fmt.Sprintf("(?s)^(%s)(?:\\s+(.+))?\\s*$",
				strings.Join(commandStrings, "|")))
	}
	{
		if err := gclient.Init(TLD); err != nil {
			log.Panicln(err)
		}
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
				m := reCommand.FindStringSubmatch(ev.Text)
				if m == nil {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"commands: "+helpString, ev.Channel))
					continue Loop
				}
				cmd := getCommandByName(m[1])
				if cmd == nil {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"commands: "+helpString, ev.Channel))
					continue Loop
				}
				u := getCreateUser(ev.User)
				if u.Level < cmd.RequiredLevel {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf(
							"unprivileged. your level: %d. required: %d",
							u.Level, cmd.RequiredLevel),
						ev.Channel))
					continue Loop
				}
				if cmd.Price > u.Points {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf(
							"not enough points. your points: %d. required: %d",
							u.Points, cmd.Price),
						ev.Channel))
					continue Loop
				}
				r := cmd.Func(m[2], u, rtm)
				if r.Text != "" {
					rtm.SendMessage(rtm.NewOutgoingMessage(r.Text, ev.Channel))
				}
				if cmd.Price > 0 && r.Charge {
					u.Points.Sub(cmd.Price)
					bank.Points.Add(cmd.Price)
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
				{
					us, _ := rtm.GetUsers()
					for _, o := range us {
						if bank.Points == 0 ||
							o.IsBot ||
							o.ID == "USLACKBOT" ||
							o.Presence != "active" {
							continue
						}
						uo := getCreateUser(o.ID)
						bank.Points.Sub(1)
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
				writeDump("./users.json", users)
				writeDump("./commands.json", commands)
				writeDump("./bank.json", bank)
			}
		}
	}
}
