package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/alfredxing/calc/compute"
	"github.com/henkman/google"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
)

type Response struct {
	Text string
}

var (
	reCommand = regexp.MustCompile("^(\\S+)\\s*(.*)$")
	cvm       *otto.Otto
	cvmWLock  sync.Mutex
	gclient   *google.Client
	commands  = map[string]func(text string) Response{
		"calc": func(text string) Response {
			if text == "" {
				return Response{
					Text: `A Calculator
Usage:
  Operators: +, -, *, /, ^, %%
  Functions: sin, cos, tan, cot, sec, csc,
             asin, acos, atan, acot, asec,
             acsc, sqrt, log, lg, ln, abs
  Constants: e, pi, π`,
				}
			}
			res, err := compute.Evaluate(text)
			if err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "Error:" + err.Error(),
				}
			}
			return Response{
				Text: fmt.Sprintf("%s=%g", text, res),
			}
		},
		"search": func(text string) Response {
			if text == "" {
				return Response{
					Text: "Finds stuff in the internet",
				}
			}
			const TLD = "de"
			if gclient == nil {
				var err error
				gclient, err = google.New()
				if err != nil {
					log.Println("ERROR:", err.Error())
					return Response{
						Text: "internal error",
					}
				}
				err = gclient.Init(TLD)
				if err != nil {
					log.Println("ERROR:", err.Error())
					return Response{
						Text: "internal error",
					}
				}
			}
			results, err := gclient.Search(TLD, text, "en", 5)
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if len(results) == 0 {
				return Response{
					Text: "nothing found",
				}
			}
			buf := bytes.NewBufferString("")
			for _, res := range results {
				fmt.Fprintf(buf, "%s %s\n", res.URL, res.Content)
			}
			return Response{
				Text: buf.String(),
			}
		},
		"video": func(text string) Response {
			if text == "" {
				return Response{
					Text: "finds videos",
				}
			}
			var r *http.Response
			{
				var err error
				u := fmt.Sprintf(
					"http://duckduckgo.com/v.js?q=%s&o=json&strict=1",
					url.QueryEscape(text))
				r, err = http.Get(u)
				if err != nil {
					log.Println("ERROR:", err)
					return Response{
						Text: "internal error",
					}
				}
				defer r.Body.Close()
			}
			var ytr struct {
				Results []struct {
					Provider string `json:"provider"`
					ID       string `json:"id"`
				} `json:"results"`
			}
			if err := json.NewDecoder(r.Body).Decode(&ytr); err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			if len(ytr.Results) == 0 {
				return Response{
					Text: "nothing found",
				}
			}
			ids := make([]string, 0, len(ytr.Results))
			for _, v := range ytr.Results {
				if v.Provider == "YouTube" {
					ids = append(ids, v.ID)
				}
			}
			if len(ids) == 0 {
				return Response{
					Text: "nothing found",
				}
			}
			o := rand.Int31n(int32(len(ytr.Results)))
			return Response{
				Text: "https://www.youtube.com/watch?v=" + ids[o],
			}
		},
		"coin": func(text string) Response {
			return Response{
				Text: map[int32]string{
					0: "heads",
					1: "tails",
				}[rand.Int31n(2)],
			}
		},
		"js": func(text string) (r Response) {
			if text == "" {
				return Response{
					Text: "interactive javascript console\nType reload to reload the VM",
				}
			}
			cvmWLock.Lock()
			defer cvmWLock.Unlock()
			if text == "reload" {
				cvm = otto.New()
				cvm.Set("console", otto.UndefinedValue())
				return Response{
					Text: "VM reloaded",
				}
			}
			if cvm == nil {
				cvm = otto.New()
				cvm.Set("console", otto.UndefinedValue())
			}
			cvm.Interrupt = make(chan func(), 1)
			defer func() {
				if timeout := recover(); timeout != nil {
					r = Response{
						Text: "Code took too long",
					}
				}
			}()
			go func() {
				time.Sleep(time.Second)
				cvm.Interrupt <- func() {
					panic("timeout")
				}
			}()
			val, err := cvm.Run(text)
			if err != nil {
				r = Response{
					Text: err.Error(),
				}
			} else {
				r = Response{
					Text: val.String(),
				}
			}
			return
		},
		"squirrel": func(text string) Response {
			return duckduckgoImage("squirrel+images", 1000)
		},
		"image": func(text string) Response {
			if text == "" {
				return Response{
					Text: "finds images",
				}
			}
			return duckduckgoImage(text, 1000)
		},
		"random": func(text string) Response {
			if text == "" {
				return Response{
					Text: "randomly prints one of the comma separated texts given",
				}
			}
			c := strings.Split(text, ",")
			if len(c) == 1 {
				return Response{
					Text: strings.TrimSpace(c[0]),
				}
			}
			t := c[rand.Int31n(int32(len(c)))]
			return Response{
				Text: strings.TrimSpace(t),
			}
		},
	}
	commandsString = func() string {
		cmds := make([]string, 0, len(commands))
		for key, _ := range commands {
			cmds = append(cmds, key)
		}
		return strings.Join(cmds, ", ")
	}()
)

func duckduckgoImage(query string, max int32) Response {
	var r *http.Response
	{
		var err error
		u := fmt.Sprintf(
			"http://duckduckgo.com/i.js?o=json&q=%s&s=%d",
			url.QueryEscape(query),
			rand.Int31n(max))
		r, err = http.Get(u)
		if err != nil {
			log.Println("ERROR:", err)
			return Response{
				Text: "nothing found",
			}
		}
		defer r.Body.Close()
	}
	var ytr struct {
		Results []struct {
			Image string `json:"image"`
		} `json:"results"`
	}
	if err := json.NewDecoder(r.Body).Decode(&ytr); err != nil {
		log.Println("ERROR:", err)
		return Response{
			Text: "nothing found",
		}
	}
	if len(ytr.Results) == 0 {
		return Response{
			Text: "nothing found",
		}
	}
	o := rand.Int31n(int32(len(ytr.Results)))
	return Response{
		Text: ytr.Results[o].Image,
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
		f, err := os.OpenFile("./log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0750)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	rand.Seed(time.Now().UnixNano())
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
				t := strings.TrimSpace(ev.Text)
				s := fmt.Sprintf("<@%s>", rtm.GetInfo().User.ID)
				if strings.Index(t, s) != 0 {
					continue Loop
				}
				t = strings.TrimLeftFunc(t[len(s):], unicode.IsSpace)
				log.Println(ev.User, t)
				m := reCommand.FindStringSubmatch(t)
				if m == nil {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"commands: "+commandsString, ev.Channel))
					continue Loop
				}
				cmd, ok := commands[m[1]]
				if !ok {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"commands: "+commandsString, ev.Channel))
					continue Loop
				}
				r := cmd(m[2])
				rtm.SendMessage(rtm.NewOutgoingMessage(r.Text, ev.Channel))
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
