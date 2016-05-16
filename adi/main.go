package main

import (
	"bytes"
	"database/sql"
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
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alfredxing/calc/compute"
	"github.com/henkman/google"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
)

type Response struct {
	Text   string
	Charge bool
}

type Level uint8

type User struct {
	ID     string
	Level  Level
	Points uint64
}

type CommandFunc func(text string, u User, rtm *slack.RTM) Response

type Command struct {
	Names         []string
	RequiredLevel Level
	Price         uint64
	Func          CommandFunc
}

const (
	TLD = "de"
)

var (
	db             *sql.DB
	defaultLevel   Level
	cvm            *otto.Otto
	gclient        google.Client
	commandMap     = map[string]Command{}
	commandStrings = func() []string {
		cmds := make([]string, 0, len(commands))
		for _, cmd := range commands {
			for _, name := range cmd.Names {
				cmds = append(cmds, name)
			}
		}
		sort.Sort(sort.StringSlice(cmds))
		return cmds
	}()
	helpString = strings.Join(commandStrings, ", ")
)

var commands = []Command{
	{
		Names:         []string{"rank"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			us, err := getUsersSortByPoints()
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			sus, err := rtm.GetUsers()
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			s := bytes.NewBufferString("")
			for i, o := range us {
				for _, su := range sus {
					if su.ID == o.ID {
						fmt.Fprintf(s, "%d. %s (%d)\n",
							i+1, su.Name, o.Points)
					}
				}
			}
			return Response{
				Text:   s.String(),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"setpts"},
		RequiredLevel: 100,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: "set points of user",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return Response{
					Text: "syntax: setpoints [username] [points]",
				}
			}
			p, err := strconv.ParseUint(s[1], 10, 64)
			if err != nil {
				return Response{
					Text: "syntax: setpoints [username] [points]",
				}
			}
			us, err := getUserByName(rtm, s[0])
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if us == nil {
				return Response{
					Text: "user not found",
				}
			}
			up := getCreateUser(us.ID)
			up.Points = p
			updateUser(up)
			return Response{
				Text: fmt.Sprintf("%s points are now %d",
					us.Name, up.Points),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"setlvl"},
		RequiredLevel: 100,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: "set level of user",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return Response{
					Text: "syntax: setlevel [username] [level]",
				}
			}
			l, err := strconv.ParseUint(s[1], 10, 8)
			if err != nil {
				return Response{
					Text: "syntax: setlevel [username] [level]",
				}
			}
			us, err := getUserByName(rtm, s[0])
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if us == nil {
				return Response{
					Text: "user not found",
				}
			}
			up := getCreateUser(us.ID)
			up.Level = Level(l)
			updateUser(up)
			return Response{
				Text: fmt.Sprintf("%s level is now %d",
					us.Name, up.Level),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"cost"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: "find out the price of a command",
				}
			}
			cmd, ok := commandMap[text]
			if !ok {
				return Response{
					Text: "command not found",
				}
			}
			return Response{
				Text:   fmt.Sprintf("%s costs %d", text, cmd.Price),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"givepts"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: "give points to username",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return Response{
					Text: "syntax: givepoints [username] [points|all]",
				}
			}
			un := s[0]
			var n uint64
			if s[1] == "all" {
				n = u.Points
			} else {
				t, err := strconv.ParseUint(s[1], 10, 64)
				if err != nil {
					return Response{
						Text: "syntax: givepoints [username] [points|all]",
					}
				}
				if t > u.Points {
					return Response{
						Text: fmt.Sprintf(
							"not enough points. your points: %d",
							u.Points),
					}
				}
				n = t
			}
			if n == 0 {
				return Response{
					Text: "Must be more than 0 points",
				}
			}
			us, err := getUserByName(rtm, un)
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if us == nil {
				return Response{
					Text: "user not found",
				}
			}
			if us.ID == u.ID {
				return Response{
					Text: "can't give points to yourself",
				}
			}
			up := getCreateUser(us.ID)
			up.Points += n
			u.Points -= n
			updateUser(u)
			updateUser(up)
			return Response{
				Text: fmt.Sprintf("%s points %d. your points: %d",
					us.Name, up.Points, u.Points),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"bet"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: "bet points to double them",
				}
			}
			var n uint64
			if text == "all" {
				n = u.Points
			} else {
				t, err := strconv.ParseUint(text, 10, 64)
				if err != nil {
					return Response{
						Text: "syntax: roulette [points|all]",
					}
				}
				if t > u.Points {
					return Response{
						Text: fmt.Sprintf(
							"not enough points. your points: %d",
							u.Points),
					}
				}
				n = t
			}
			if n == 0 {
				return Response{
					Text: "Must be more than 0 points",
				}
			}
			var t string
			if rand.Int31n(2) == 1 {
				n *= 2
				u.Points += n
				t = fmt.Sprintf("You won %d points. New points %d",
					n, u.Points)
			} else {
				u.Points -= n
				t = fmt.Sprintf("You lost %d points. New points %d",
					n, u.Points)
			}
			updateUser(u)
			return Response{
				Text:   t,
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"pts"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: fmt.Sprintf("your points: %d", u.Points),
				}
			}
			us, err := getUserByName(rtm, text)
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if us == nil {
				return Response{
					Text: "user not found",
				}
			}
			up := getCreateUser(us.ID)
			return Response{
				Text:   fmt.Sprintf("%s points: %d", us.Name, up.Points),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"lvl"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: fmt.Sprintf("your level: %d", u.Level),
				}
			}
			us, err := getUserByName(rtm, text)
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if us == nil {
				return Response{
					Text: "user not found",
				}
			}
			up := getCreateUser(us.ID)
			return Response{
				Text:   fmt.Sprintf("%s level: %d", us.Name, up.Level),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"id"},
		RequiredLevel: 10,
		Price:         0,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: fmt.Sprintf("your id: %s", u.ID),
				}
			}
			us, err := getUserByName(rtm, text)
			if err != nil {
				log.Println("ERROR:", err.Error())
				return Response{
					Text: "internal error",
				}
			}
			if us == nil {
				return Response{
					Text: "user not found",
				}
			}
			return Response{
				Text:   fmt.Sprintf("%s id: %s", us.Name, us.ID),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"calc"},
		RequiredLevel: 10,
		Price:         5,
		Func: func(text string, u User, rtm *slack.RTM) Response {
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
				Text:   fmt.Sprintf("%s=%g", text, res),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"web"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
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
				Text:   buf.String(),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"vid"},
		RequiredLevel: 10,
		Price:         30,
		Func: func(text string, u User, rtm *slack.RTM) Response {
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
				Text:   "https://www.youtube.com/watch?v=" + ids[o],
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"coin"},
		RequiredLevel: 10,
		Price:         1,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return Response{
				Text: []string{
					"heads",
					"tails",
				}[rand.Int31n(2)],
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"js"},
		RequiredLevel: 10,
		Price:         50,
		Func: func(text string, u User, rtm *slack.RTM) (r Response) {
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
					Text:   val.String(),
					Charge: true,
				}
			}
			return
		},
	},
	{
		Names:         []string{"img"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return googleImage(text, true, google.ImageType_Any)
		},
	},
	{
		Names:         []string{"gif"},
		RequiredLevel: 10,
		Price:         30,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return googleImage(text, true, google.ImageType_Animated)
		},
	},
	{
		Names:         []string{"nsfwimg"},
		RequiredLevel: 80,
		Price:         100,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return googleImage(text, false, google.ImageType_Any)
		},
	},
	{
		Names:         []string{"nsfwgif"},
		RequiredLevel: 80,
		Price:         100,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return googleImage(text, false, google.ImageType_Animated)
		},
	},
	{
		Names:         []string{"bikpin"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			const N = 400
			return duckduckgoImage("bikini+pineapple", uint(rand.Int31n(N)))
		},
	},
	{
		Names:         []string{"squirl"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			const N = 1000
			return duckduckgoImage("squirrel+images", uint(rand.Int31n(N)))
		},
	},
	{
		Names:         []string{"rndimg"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			const N = 1000
			if text == "" {
				return Response{
					Text: fmt.Sprintf(
						"gets random image from first %d search results", N),
				}
			}
			return duckduckgoImage(text, uint(rand.Int31n(N)))
		},
	},
	{
		Names:         []string{"rnd"},
		RequiredLevel: 10,
		Price:         2,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			if text == "" {
				return Response{
					Text: "randomly prints one of the comma separated texts given",
				}
			}
			c := strings.Split(text, ",")
			if len(c) == 1 {
				return Response{
					Text:   strings.TrimSpace(c[0]),
					Charge: true,
				}
			}
			t := c[rand.Int31n(int32(len(c)))]
			return Response{
				Text:   strings.TrimSpace(t),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"mpoll"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return poll(text, true)
		},
	},
	{
		Names:         []string{"spoll"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return poll(text, false)
		},
	},
	{
		Names:         []string{"tr"},
		RequiredLevel: 10,
		Price:         2,
		Func: func(text string, u User, rtm *slack.RTM) Response {
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
	},
	{
		Names:         []string{"en"},
		RequiredLevel: 10,
		Price:         2,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return googleTranslate(text, "en")
		},
	},
	{
		Names:         []string{"de"},
		RequiredLevel: 10,
		Price:         2,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			return googleTranslate(text, "de")
		},
	},
	{
		Names:         []string{"fact"},
		RequiredLevel: 10,
		Price:         5,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			doc, err := goquery.NewDocument("http://randomfunfacts.com/")
			if err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			return Response{
				Text:   doc.Find("center i").Text(),
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"toon"},
		RequiredLevel: 10,
		Price:         10,
		Func: func(text string, u User, rtm *slack.RTM) Response {
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
				Text:   img,
				Charge: true,
			}
		},
	},
	{
		Names:         []string{"insult"},
		RequiredLevel: 10,
		Price:         5,
		Func: func(text string, u User, rtm *slack.RTM) Response {
			doc, err := goquery.NewDocument("http://www.randominsults.net/")
			if err != nil {
				log.Println("ERROR:", err)
				return Response{
					Text: "internal error",
				}
			}
			return Response{
				Text:   doc.Find("center i").Text(),
				Charge: true,
			}
		},
	},
}

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
		Text:   fmt.Sprintf("%s: %s", l, t),
		Charge: true,
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
		Text:   images[r].URL,
		Charge: true,
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
		Text:   p,
		Charge: true,
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
		Text:   ytr.Results[o].Image,
		Charge: true,
	}
}

func getUserByName(rtm *slack.RTM, name string) (*slack.User, error) {
	us, err := rtm.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, o := range us {
		if o.Name == name {
			return &o, nil
		}
	}
	return nil, nil
}

func getUsersSortByPoints() ([]User, error) {
	r, err := db.Query("SELECT id, level, points FROM user ORDER BY points desc")
	if err != nil {
		return nil, err
	}
	us := make([]User, 0, 10)
	for r.Next() {
		var u User
		r.Scan(&u.ID, &u.Level, &u.Points)
		us = append(us, u)
	}
	r.Close()
	return us, nil
}

func updateUser(user User) {
	db.Exec("UPDATE user SET level=?, points=? WHERE id=?",
		user.Level, user.Points, user.ID)
}

func getCreateUser(id string) User {
	var points uint64
	var level Level
	if err := db.QueryRow(
		"SELECT level, points FROM user WHERE id=?",
		id).Scan(&level, &points); err != nil {
		user := User{ID: id, Level: defaultLevel, Points: 0}
		db.Exec("INSERT into user(id, level, points) values(?, ?, ?)",
			user.ID, user.Level, user.Points)
		return user
	}
	return User{ID: id, Level: level, Points: points}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	{
		for _, cmd := range commands {
			for _, name := range cmd.Names {
				commandMap[name] = cmd
			}
		}
	}
	{
		f, err := os.OpenFile("./log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0750)
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
		fd, err := os.OpenFile("./config.json", os.O_RDONLY, 0600)
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
		d, err := sql.Open("sqlite3", "./slack.db")
		if err != nil {
			log.Panicln(err)
		}
		defer d.Close()
		db = d
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
				cmd, ok := commandMap[m[1]]
				if !ok {
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
					if cmd.Price > u.Points {
						u.Points = 0
					} else {
						u.Points -= cmd.Price
					}
					updateUser(u)
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
