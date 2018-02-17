package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
)

type Order struct {
	Number int
	Extra  string
}

type Orderer struct {
	ID     string
	Orders []Order
}

func getUserById(users []slack.User, id string) *slack.User {
	for i, o := range users {
		if o.ID == id {
			return &users[i]
		}
	}
	return nil
}

func getOrdererById(orderers []Orderer, id string) *Orderer {
	for i, o := range orderers {
		if o.ID == id {
			return &orderers[i]
		}
	}
	return nil
}

func main() {
	orderers := []Orderer{}
	var config struct {
		Key string `json:"key"`
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
						`^<@%s(?:|[^>]+)?>\s*(listall|clearall|summary|order|list|remove|clear)?\s*(\d+)?\s*(.*)$`,
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
					orderer := getOrdererById(orderers, ev.User)
					if orderer == nil {
						orderers = append(orderers, Orderer{
							ID:     ev.User,
							Orders: []Order{order},
						})
					} else {
						orderer.Orders = append(orderer.Orders, order)
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf("your order(%d %s) has been added",
							order.Number, order.Extra), ev.Channel))
				case "list":
					orderer := getOrdererById(orderers, ev.User)
					if orderer == nil || len(orderer.Orders) == 0 {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"you currently have no order", ev.Channel))
						break
					}
					if len(orderer.Orders) == 1 {
						order := orderer.Orders[0]
						var msg string
						if order.Extra != "" {
							msg = fmt.Sprintf("your order: %d(%s)\n",
								order.Number, order.Extra)
						} else {
							msg = fmt.Sprintf("your order: %d\n",
								order.Number)
						}
						rtm.SendMessage(rtm.NewOutgoingMessage(msg, ev.Channel))
						break
					}
					var buffer strings.Builder
					buffer.WriteString(fmt.Sprintf("your current orders:\n"))
					for i, order := range orderer.Orders {
						if order.Extra != "" {
							buffer.WriteString(
								fmt.Sprintf("\t%d. %d(%s)\n",
									i, order.Number, order.Extra))
						} else {
							buffer.WriteString(
								fmt.Sprintf("\t%d. %d\n",
									i, order.Number))
						}
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(
						buffer.String(), ev.Channel))
				case "remove":
					if m[2] == "" {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"syntax: remove number", ev.Channel))
						continue Loop
					}
					num, err := strconv.Atoi(m[2])
					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"only numbers are allowed", ev.Channel))
						continue Loop
					}
					orderer := getOrdererById(orderers, ev.User)
					if orderer == nil || len(orderer.Orders) == 0 {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"you have no order", ev.Channel))
						break
					}
					if num >= len(orderer.Orders) {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"order does not exist", ev.Channel))
						break
					}
					orderer.Orders = append(orderer.Orders[:num],
						orderer.Orders[num+1:]...)
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"order was removed", ev.Channel))
				case "clear":
					orderer := getOrdererById(orderers, ev.User)
					if orderer == nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"you have no order", ev.Channel))
					} else {
						orderer.Orders = orderer.Orders[:0]
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"orders cleared", ev.Channel))
					}
				case "listall":
					users, err := rtm.GetUsers()
					if err != nil {
						log.Println("ERROR:", "couldn't get users")
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"internal error", ev.Channel))
						continue Loop
					}
					var buffer strings.Builder
					buffer.WriteString("current orders:\n")
					for _, orderer := range orderers {
						user := getUserById(users, orderer.ID)
						if user == nil || len(orderer.Orders) == 0 {
							continue
						}
						if len(orderer.Orders) == 1 {
							order := orderer.Orders[0]
							if order.Extra != "" {
								buffer.WriteString(
									fmt.Sprintf("\t%s - %d(%s)\n",
										user.Name, order.Number, order.Extra))
							} else {
								buffer.WriteString(
									fmt.Sprintf("\t%s - %d\n",
										user.Name, order.Number))
							}
							continue
						}
						buffer.WriteString(fmt.Sprintf("\t%s - ", user.Name))
						for i, order := range orderer.Orders {
							if order.Extra != "" {
								buffer.WriteString(
									fmt.Sprintf("%d(%s)",
										order.Number, order.Extra))
							} else {
								buffer.WriteString(
									fmt.Sprintf("%d", order.Number))
							}
							if i != len(orderer.Orders)-1 {
								buffer.WriteString(", ")
							}
						}
						buffer.WriteByte('\n')
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(
						strings.TrimSpace(buffer.String()), ev.Channel))
				case "summary":
					users, err := rtm.GetUsers()
					if err != nil {
						log.Println("ERROR:", "couldn't get users")
						rtm.SendMessage(rtm.NewOutgoingMessage(
							"internal error", ev.Channel))
						continue Loop
					}

					var buffer strings.Builder
					buffer.WriteString("current orders:\n")
					for _, orderer := range orderers {
						user := getUserById(users, orderer.ID)
						if user == nil || len(orderer.Orders) == 0 {
							continue
						}

						groupedOrders := make(map[int]int)
						for _, order := range orderer.Orders {
							groupedOrders[order.Number] = groupedOrders[order.Number] + 1
						}

						var numbers []int
						for number := range groupedOrders {
							numbers = append(numbers, number)
						}
						sort.Ints(numbers)

						for _, number := range numbers {
							buffer.WriteString(
								fmt.Sprintf("%dx %d", groupedOrders[number], number))
							buffer.WriteByte('\n')
						}
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(
						strings.TrimSpace(buffer.String()), ev.Channel))
				case "clearall":
					orderers = orderers[:0]
					rtm.SendMessage(rtm.NewOutgoingMessage("orderers cleared",
						ev.Channel))
				default:
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"*commands:* order, list, remove, clear",
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
