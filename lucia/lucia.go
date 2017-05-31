package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
)

type Order struct {
	Number int
	Extra  string
}

func getUserById(users []slack.User, id string) *slack.User {
	for i, o := range users {
		if o.ID == id {
			return &users[i]
		}
	}
	return nil
}

func main() {
	orders := make(map[string]Order)
	var config struct {
		Key    string `json:"key"`
		PdfUrl string `json:"pdf_url"`
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
	var reCommand *regexp.Regexp
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
				reCommand = regexp.MustCompile(
					fmt.Sprintf(
						"^<@%s(?:|[^>]+)?>\\s*(show|order|cancel|list|clear|card)?\\s*(\\d+)?\\s*(.*)$",
						rtm.GetInfo().User.ID))
			case *slack.MessageEvent:
				m := reCommand.FindStringSubmatch(ev.Text)
				if m == nil {
					continue Loop
				}
				switch m[1] {
				case "order":
					if m[2] == "" {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"syntax: order number (extras)", ev.Channel))
						continue Loop
					}
					num, err := strconv.Atoi(m[2])
					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"only numbers are allowed", ev.Channel))
						continue Loop
					}
					order := Order{
						Number: num,
						Extra:  m[3],
					}
					orders[ev.User] = order
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf("your order(%d %s) has been added",
							order.Number, order.Extra), ev.Channel))
				case "cancel":
					if _, ok := orders[ev.User]; ok {
						delete(orders, ev.User)
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"your order has been canceled", ev.Channel))
					} else {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"you currently have no order", ev.Channel))
					}
				case "show":
					if order, ok := orders[ev.User]; ok {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							fmt.Sprintf("your current order: %d(%s)",
								order.Number, order.Extra), ev.Channel))
					} else {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"you currently have no order", ev.Channel))
					}
				case "list":
					users, err := rtm.GetUsers()
					if err != nil {
						log.Println("ERROR:", "couldn't get users")
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"internal error", ev.Channel))
						continue Loop
					}
					var buffer bytes.Buffer
					buffer.WriteString(strconv.Itoa(len(orders)))
					if len(orders) == 1 {
						buffer.WriteString(" order was made:\n\n")
					} else {
						buffer.WriteString(" orders were made:\n\n")
					}
					for userid, order := range orders {
						user := getUserById(users, userid)
						if user == nil {
							continue
						}
						if order.Extra != "" {
							buffer.WriteString(
								fmt.Sprintf("%s - %d(%s)\n",
									user, order.Number, order.Extra))
						} else {
							buffer.WriteString(
								fmt.Sprintf("%s - %d\n", user, order.Number))
						}
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(
						strings.TrimSpace(buffer.String()), ev.Channel))
				case "clear":
					orders = make(map[string]Order)
					rtm.SendMessage(rtm.NewOutgoingMessage("list cleared", ev.Channel))
				case "card":
					rtm.SendMessage(rtm.NewOutgoingMessage(config.PdfUrl, ev.Channel))
				default:
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"*commands:* order number (extrawish), cancel, list, clear, card",
						ev.Channel))
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
		}
	}
}
