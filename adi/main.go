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
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alfredxing/calc/compute"
	"github.com/henkman/google"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
)

type Response struct {
	Text string
}

const (
	TLD = "de"
)

var (
	cvm      *otto.Otto
	gclient  google.Client
	commands = map[string]func(text string) Response{
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
		"web": func(text string) Response {
			if text == "" {
				return Response{
					Text: "Finds stuff in the internet",
				}
			}
			results, err := gclient.Search(TLD, text, "en", false, 5)
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
		"vid": func(text string) Response {
			if text == "" {
				return Response{
					Text: "finds videos",
				}
			}
			var r *http.Response
			{
				var err error
				u := fmt.Sprintf(
					"https://duckduckgo.com/v.js?q=%s&o=json&strict=1",
					url.QueryEscape(text))
				r, err = http.Get(u)
				if err != nil {
					log.Println("ERROR:", err)
					return Response{
						Text: "internal error",
					}
				}
			}
			var ytr struct {
				Results []struct {
					Provider string `json:"provider"`
					ID       string `json:"id"`
				} `json:"results"`
			}
			if err := json.NewDecoder(r.Body).Decode(&ytr); err != nil {
				r.Body.Close()
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			r.Body.Close()
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
				Text: []string{
					"heads",
					"tails",
				}[rand.Int31n(2)],
			}
		},
		"js": func(text string) (r Response) {
			if text == "" {
				return Response{
					Text: "interactive javascript console\nType reload to reload the VM",
				}
			}
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
		"img": func(text string) Response {
			return googleImage(text, true, google.ImageType_Any)
		},
		"gif": func(text string) Response {
			return googleImage(text, true, google.ImageType_Animated)
		},
		"bikpin": func(text string) Response {
			const N = 250
			return duckduckgoImage("bikini+pineapple", uint(rand.Int31n(N)))
		},
		"squirrel": func(text string) Response {
			const N = 1000
			return duckduckgoImage("squirrel+images", uint(rand.Int31n(N)))
		},
		"randimg": func(text string) Response {
			const N = 1000
			if text == "" {
				return Response{
					Text: fmt.Sprintf(
						"gets random image from first %d search results", N),
				}
			}
			return duckduckgoImage(text, uint(rand.Int31n(N)))
		},
		"rand": func(text string) Response {
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
		"multipoll": func(text string) Response {
			return poll(text, true)
		},
		"singlepoll": func(text string) Response {
			return poll(text, false)
		},
		"tr": func(text string) Response {
			languages := []string{
				"af", "ar", "az", "be", "bg", "ca", "cs", "cy", "da", "de",
				"el", "en", "es", "et", "eu", "fa", "fi", "fr", "ga", "gl",
				"hi", "hr", "ht", "hu", "hy", "id", "is", "it", "iw", "ja",
				"ka", "ko", "lt", "lv", "mk", "ms", "mt", "nl", "no", "pl",
				"pt", "ro", "ru", "sk", "sl", "sq", "sr", "sv", "sw", "th",
				"tl", "tr", "uk", "ur", "vi", "yi",
			}
			help := "translates text. available languages:\n" +
				strings.Join(languages, ", ")
			if text == "" {
				return Response{
					Text: help,
				}
			}
			s := strings.Index(text, " ")
			if s == -1 {
				return Response{
					Text: help,
				}
			}
			l := text[:s]
			{
				ok := false
				for _, e := range languages {
					if e == l {
						ok = true
						break
					}
				}
				if !ok {
					return Response{
						Text: "language not supported",
					}
				}
			}
			t := text[s:]
			return googleTranslate(t, l)
		},
		"en": func(text string) Response {
			return googleTranslate(text, "en")
		},
		"de": func(text string) Response {
			return googleTranslate(text, "de")
		},
		"fact": func(text string) Response {
			doc, err := goquery.NewDocument("http://randomfunfacts.com/")
			if err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			return Response{
				Text: doc.Find("center i").Text(),
			}
		},
		"cartoon": func(text string) Response {
			doc, err := goquery.NewDocument("http://www.veryfunnycartoons.com/")
			if err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			img, ok := doc.Find("center i img").Attr("src")
			if !ok {
				log.Println("ERROR: cartoon img src not found")
				return Response{
					Text: "internal error",
				}
			}
			return Response{
				Text: img,
			}
		},
		"insult": func(text string) Response {
			doc, err := goquery.NewDocument("http://www.randominsults.net/")
			if err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			return Response{
				Text: doc.Find("center i").Text(),
			}
		},
	}
	commandStrings = func() []string {
		cmds := make([]string, 0, len(commands))
		for key, _ := range commands {
			cmds = append(cmds, key)
		}
		sort.Sort(sort.StringSlice(cmds))
		return cmds
	}()
	helpString = strings.Join(commandStrings, ", ")
)

func googleTranslate(text, tl string) Response {
	r, err := http.PostForm("https://translate.google.com/translate_a/t",
		url.Values{
			"client": []string{"x"},
			"hl":     []string{"en"},
			"sl":     []string{"auto"},
			"text":   []string{text},
			"tl":     []string{tl},
		})
	if err != nil {
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	tj := make([]string, 0, 2)
	if err := json.NewDecoder(r.Body).Decode(&tj); err != nil {
		r.Body.Close()
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	r.Body.Close()
	if len(tj) != 2 {
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	t, l := tj[0], tj[1]
	return Response{
		Text: fmt.Sprintf("%s: %s", l, t),
	}
}

func googleImage(text string, safe bool, typ google.ImageType) Response {
	if text == "" {
		return Response{
			Text: "finds images",
		}
	}
	images, err := gclient.Images(TLD, text, "de", safe, typ, 50)
	if err != nil {
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	if len(images) == 0 {
		return Response{
			Text: "nothing found",
		}
	}
	r := rand.Int31n(int32(len(images)))
	return Response{
		Text: images[r].URL,
	}
}

func poll(text string, multi bool) Response {
	if text == "" {
		return Response{
			Text: `Creates a poll
Example: poll animal?, dog, cat, hamster
-> Creates a poll with title animal? and the three animals as choices`,
		}
	}
	s := strings.Split(text, ",")
	if len(s) < 3 {
		return Response{
			Text: "Needs one question and at least 2 options",
		}
	}
	preq := struct {
		Title    string   `json:"title"`
		Options  []string `json:"options"`
		Multi    bool     `json:"multi"`
		Dupcheck string   `json:"dupcheck"`
	}{
		s[0],
		s[1:],
		multi,
		"permissive",
	}
	data, err := json.Marshal(&preq)
	if err != nil {
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	r, err := http.Post("https://www.strawpoll.me/api/v2/polls",
		"application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	var pres struct {
		ID uint64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&pres); err != nil {
		r.Body.Close()
		log.Println("ERROR:", err)
		return Response{
			Text: "internal error",
		}
	}
	r.Body.Close()
	p := fmt.Sprintf("http://www.strawpoll.me/%d", pres.ID)
	log.Println("new poll:", p)
	return Response{
		Text: p,
	}
}

func duckduckgoImage(query string, offset uint) Response {
	var r *http.Response
	{
		u := "https://duckduckgo.com/i.js?o=json&q=" + url.QueryEscape(query)
		if offset > 0 {
			u += fmt.Sprintf("&s=%d", offset)
		}
		var err error
		r, err = http.Get(u)
		if err != nil {
			log.Println("ERROR:", err)
			return Response{
				Text: "nothing found",
			}
		}
	}
	var ytr struct {
		Results []struct {
			Image string `json:"image"`
		} `json:"results"`
	}
	if err := json.NewDecoder(r.Body).Decode(&ytr); err != nil {
		r.Body.Close()
		log.Println("ERROR:", err)
		return Response{
			Text: "nothing found",
		}
	}
	r.Body.Close()
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
		Key              string `json:"key"`
		ShortCommands    bool   `json:"short_commands"`
		ShortCommandSign string `json:"short_command_sign"`
	}
	{
		fd, err := os.OpenFile("./config.json", os.O_RDONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.NewDecoder(fd).Decode(&config); err != nil {
			fd.Close()
			log.Fatal(err)
		}
		fd.Close()
	}
	{
		f, err := os.OpenFile("./log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0750)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	{
		if err := gclient.Init(TLD); err != nil {
			log.Fatal(err)
		}
	}
	var (
		reToMe    *regexp.Regexp
		reCommand *regexp.Regexp
	)
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
				if config.ShortCommands {
					reToMe = regexp.MustCompile(fmt.Sprintf("^(?:<@%s>\\s*|%s)",
						rtm.GetInfo().User.ID,
						config.ShortCommandSign))
				} else {
					reToMe = regexp.MustCompile(fmt.Sprintf("^<@%s>\\s*",
						rtm.GetInfo().User.ID))
				}
				reCommand = regexp.MustCompile(
					fmt.Sprintf("(?s)^(%s)\\s*(.*)\\s*$",
						strings.Join(commandStrings, "|")))
			case *slack.MessageEvent:
				{
					m := reToMe.FindStringSubmatch(ev.Text)
					if m == nil {
						continue Loop
					}
					ev.Text = ev.Text[len(m[0]):]
				}
				m := reCommand.FindStringSubmatch(ev.Text)
				if m == nil {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"commands: "+helpString, ev.Channel))
					continue Loop
				}
				cmd, ok := commands[m[1]]
				if !ok {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						"commands: "+helpString, ev.Channel))
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
