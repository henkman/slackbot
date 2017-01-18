package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

func main() {
	var config struct {
		Key       string   `json:"key"`
		Sentences []string `json:"sentences"`
	}
	{
		fd, err := os.OpenFile("./config.json", os.O_RDONLY, 0600)
		if err != nil {
			panic(err)
		}
		if err := json.NewDecoder(fd).Decode(&config); err != nil {
			fd.Close()
		}
		fd.Close()
	}
	{
		fd, err := os.OpenFile("./log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0750)
		if err != nil {
			panic(err)
		}
		defer fd.Close()
		log.SetOutput(fd)
	}
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	api := slack.New(config.Key)
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
			case *slack.MessageEvent:
				s := fmt.Sprintf("<@%s>", rtm.GetInfo().User.ID)
				if !strings.Contains(ev.Text, s) {
					continue Loop
				}
				c := config.Sentences[rand.Int31n(int32(len(config.Sentences)))]
				rtm.SendMessage(rtm.NewOutgoingMessage(c, ev.Channel))
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
