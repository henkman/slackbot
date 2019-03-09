package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

type Command func(text string, m *slack.MessageEvent, rtm *slack.RTM)

var (
	config struct {
		Debug            bool   `json:"debug"`
		Key              string `json:"key"`
		ShortCommands    bool   `json:"short_commands"`
		ShortCommandSign string `json:"short_command_sign"`
		Google           struct {
			Key string `json:"key"`
			CSE string `json:"cse"`
		} `json:"google"`
	}
	client = http.Client{
		Timeout: time.Second * 10,
	}
	commands = map[string]Command{
		"img": img,
	}
)

func readConfig() {
	fd, err := os.OpenFile("./config.json", os.O_RDONLY, 0750)
	if err != nil {
		log.Panicln(err)
	}
	if err := json.NewDecoder(fd).Decode(&config); err != nil {
		fd.Close()
		log.Panicln(err)
	}
}

func main() {
	{
		f, err := os.OpenFile("./log",
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0750)
		if err != nil {
			log.Panicln(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	readConfig()
	var reCommand, reToMe *regexp.Regexp
	{
		commandStrings := make([]string, len(commands))
		i := 0
		for name, _ := range commands {
			commandStrings[i] = name
			i++
		}
		reCommand = regexp.MustCompile(
			fmt.Sprintf("(?s)^(%s)(?:\\s+(.+))?\\s*$",
				strings.Join(commandStrings, "|")))
	}
	api := slack.New(config.Key, slack.OptionDebug(config.Debug))
	rtm := api.NewRTM()
	go rtm.ManageConnection()
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectedEvent:
				log.Println("connected")
				if config.ShortCommands {
					reToMe = regexp.MustCompile(fmt.Sprintf("^(?:<@%s>\\s*|%s)",
						rtm.GetInfo().User.ID,
						config.ShortCommandSign))
				} else {
					reToMe = regexp.MustCompile(fmt.Sprintf("^<@%s>\\s*",
						rtm.GetInfo().User.ID))
				}
			case *slack.MessageEvent:
				if ev.User == rtm.GetInfo().User.ID {
					continue Loop
				}
				m := reToMe.FindStringSubmatch(ev.Text)
				if m == nil {
					continue Loop
				}
				text := ev.Text[len(m[0]):]
				m = reCommand.FindStringSubmatch(text)
				if m == nil {
					continue Loop
				}
				cmd, ok := commands[m[1]]
				if !ok {
					continue Loop
				}
				log.Println(ev.User, m[1], m[2])
				cmd(m[2], ev, rtm)
			case *slack.PresenceChangeEvent:
			case *slack.LatencyReport:
			case *slack.RTMError:
				log.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				log.Println("Invalid credentials")
				break Loop
			default:
			}
		}
	}
}
