package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
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

type User struct {
	ID     string `json:"id"`
	Level  Level  `json:"level"`
	Points uint64 `json:"points"`
}

type CommandFunc func(text string, u *User, rtm *slack.RTM) Response

type Command struct {
	Name          string      `json:"name"`
	RequiredLevel Level       `json:"required_level"`
	Price         uint64      `json:"price"`
	Func          CommandFunc `json:"-"`
}

const (
	TLD = "de"
)

var (
	defaultLevel   Level
	cvm            *otto.Otto
	gclient        google.Client
	users          []User
	commands       []Command
	commandFuncs   = map[string]CommandFunc{}
	commandStrings []string
	helpString     string
)

func (u *User) Add(p uint64) {
	if u.Points > (math.MaxUint64 - p) {
		u.Points = math.MaxUint64
	} else {
		u.Points += p
	}
}

func (u *User) Sub(p uint64) {
	if u.Points < p {
		u.Points = 0
	} else {
		u.Points -= p
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
		fd, err := os.OpenFile("./commands.json", os.O_RDONLY, 0750)
		if err != nil {
			log.Panicln(err)
		}
		commands = make([]Command, 0, 10)
		if err := json.NewDecoder(fd).Decode(&commands); err != nil {
			fd.Close()
			log.Panicln(err)
		}
		fd.Close()
		commandStrings := make([]string, 0, len(commands))
		for i, _ := range commands {
			name := commands[i].Name
			commands[i].Func = commandFuncs[name]
			commandStrings = append(commandStrings, name)
		}
		sort.Sort(sort.StringSlice(commandStrings))
		helpString = strings.Join(commandStrings, ", ")
	}
	{
		fd, err := os.OpenFile("./users.json", os.O_RDONLY, 0750)
		if err != nil {
			log.Panicln(err)
		}
		users = make([]User, 0, 10)
		if err := json.NewDecoder(fd).Decode(&users); err != nil {
			fd.Close()
			log.Panicln(err)
		}
		fd.Close()
	}
	{
		if err := gclient.Init(TLD); err != nil {
			log.Panicln(err)
		}
	}
	var (
		reToMe    *regexp.Regexp
		reCommand *regexp.Regexp
	)
	tick := time.NewTicker(time.Minute)
	rand.Seed(time.Now().UnixNano())
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
				reCommand = regexp.MustCompile(
					fmt.Sprintf("(?s)^(%s)(?:\\s+(.+))?\\s*$",
						strings.Join(commandStrings, "|")))
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
					u.Sub(cmd.Price)
				}
			case *slack.PresenceChangeEvent:
			case *slack.LatencyReport:
			case *slack.RTMError:
				log.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				log.Println("Invalid credentials")
				break Loop
			default:
			}
		case <-tick.C:
			{
				{
					us, _ := rtm.GetUsers()
					for _, o := range us {
						if o.IsBot ||
							o.ID == "USLACKBOT" ||
							o.Presence != "active" {
							continue
						}
						for i, _ := range users {
							if users[i].ID == o.ID {
								users[i].Add(1)
							}
						}
					}
				}
				{
					fd, err := os.OpenFile("./users.json",
						os.O_WRONLY|os.O_TRUNC, 0750)
					if err != nil {
						log.Println(err)
						continue Loop
					}
					if err := json.NewEncoder(fd).Encode(users); err != nil {
						fd.Close()
						log.Println(err)
						continue Loop
					}
					fd.Close()
				}
				{
					fd, err := os.OpenFile("./commands.json",
						os.O_WRONLY|os.O_TRUNC, 0750)
					if err != nil {
						log.Println(err)
						continue Loop
					}
					if err := json.NewEncoder(fd).Encode(commands); err != nil {
						fd.Close()
						log.Println(err)
						continue Loop
					}
					fd.Close()
				}
			}
		}
	}
}
