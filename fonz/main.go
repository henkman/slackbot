package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/henkman/markov"
	"github.com/nlopes/slack"
)

func WordJoin(words []string) string {
	text := ""
	for i, _ := range words {
		text += words[i]
		isLast := i == (len(words) - 1)
		if !isLast {
			next := words[i+1]
			fc := []rune(next)[0]
			word := []rune(words[i])
			lc := word[len(word)-1]
			if lc == '.' || lc == ',' || lc == '?' ||
				lc == '!' || lc == ';' ||
				(unicode.IsLetter(lc) || unicode.IsDigit(lc)) &&
					(unicode.IsLetter(fc) || unicode.IsDigit(fc)) {
				text += " "
			}
		}
	}
	return text
}

func main() {
	var config struct {
		Key       string `json:"key"`
		DeleteKey string `json:"deletekey"`
		MinWords  int    `json:"minwords"`
		MaxWords  int    `json:"maxwords"`
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
	var tg markov.TextGenerator
	tg.Init(time.Now().Unix())
	if fd, err := os.Open("seed.txt"); err == nil {
		br := bufio.NewReader(fd)
		var buf bytes.Buffer
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				break
			}
			buf.Reset()
			buf.WriteString(strings.ToLower(line[:len(line)-1]))
			tg.Feed(&buf)
		}
		fd.Close()
	}
	api := slack.New(config.Key)
	api.SetDebug(false)
	deleteapi := slack.New(config.DeleteKey)
	deleteapi.SetDebug(false)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	var reAtme *regexp.Regexp
	var myid string
	mr := rand.New(rand.NewSource(time.Now().Unix()))
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectedEvent:
				myid = rtm.GetInfo().User.ID
				reAtme = regexp.MustCompile(
					fmt.Sprintf("<@%s(?:\\|[^>]+)?>", myid))
				var buf bytes.Buffer
				feed := func(ms []slack.Message) {
					for _, m := range ms {
						if m.User == myid ||
							strings.HasPrefix(m.Text, "!") ||
							strings.Contains(m.Text, "<@") ||
							strings.Contains(m.Text, "<!") ||
							strings.Contains(m.Text, "<#") ||
							strings.Contains(m.Text, "http") {
							continue
						}
						buf.Reset()
						buf.WriteString(strings.ToLower(m.Text))
						tg.Feed(&buf)
					}
				}
				channels, err := rtm.GetChannels(true)
				if err != nil {
					log.Fatal(err)
				}
				for _, ch := range channels {
					if !ch.IsMember {
						continue
					}
					log.Println("reading channel", ch.Name)
					l := ""
					for {
						h, err := rtm.GetChannelHistory(ch.ID,
							slack.HistoryParameters{
								Count:     1000,
								Unreads:   true,
								Inclusive: false,
								Latest:    l,
							})
						if err != nil {
							log.Fatal(err)
						}
						feed(h.Messages)
						if !h.HasMore {
							break
						}
						l = h.Messages[len(h.Messages)-1].Timestamp
					}
				}
				groups, err := rtm.GetGroups(true)
				if err != nil {
					log.Fatal(err)
				}
				for _, ch := range groups {
					member := false
					for _, m := range ch.Members {
						if m == myid {
							member = true
							break
						}
					}
					if !member {
						continue
					}
					log.Println("reading group", ch.Name)
					l := ""
					for {
						h, err := rtm.GetGroupHistory(ch.ID,
							slack.HistoryParameters{
								Count:     1000,
								Unreads:   true,
								Inclusive: false,
								Latest:    l,
							})
						if err != nil {
							log.Fatal(err)
						}
						feed(h.Messages)
						if !h.HasMore {
							break
						}
						l = h.Messages[len(h.Messages)-1].Timestamp
					}
				}
				log.Println("init done")
			case *slack.MessageEvent:
				if ev.User == myid {
					continue Loop
				}
				if strings.HasPrefix(ev.Text, "!") ||
					strings.Contains(ev.Text, "<!") ||
					strings.Contains(ev.Text, "<#") ||
					strings.Contains(ev.Text, "http") {
					continue Loop
				}
				if reAtme.FindString(ev.Text) != "" {
					x := mr.Int31n(int32(config.MaxWords-config.MinWords)+1) +
						int32(config.MinWords)
					text := WordJoin(tg.Generate(uint(x)))
					rtm.SendMessage(rtm.NewOutgoingMessage(text, ev.Channel))
					deleteapi.DeleteMessage(ev.Channel, ev.Timestamp)
					continue Loop
				} else if !strings.Contains(ev.Text, "<@") {
					tg.Feed(bytes.NewBufferString(strings.ToLower(ev.Text)))
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
