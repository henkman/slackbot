package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/henkman/google"
	"github.com/nlopes/slack"
)

func init() {
	commandFuncs["web"] = func(text string, u *User, rtm *slack.RTM) Response {
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
	}
	commandFuncs["vid"] = func(text string, u *User, rtm *slack.RTM) Response {
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
	}
	commandFuncs["img"] = func(text string, u *User, rtm *slack.RTM) Response {
		return googleImage(text, true, google.ImageType_Any)
	}
	commandFuncs["gif"] = func(text string, u *User, rtm *slack.RTM) Response {
		return googleImage(text, true, google.ImageType_Animated)
	}
	commandFuncs["nsfwimg"] = func(text string, u *User, rtm *slack.RTM) Response {
		return googleImage(text, false, google.ImageType_Any)
	}
	commandFuncs["nsfwgif"] = func(text string, u *User, rtm *slack.RTM) Response {
		return googleImage(text, false, google.ImageType_Animated)
	}
	commandFuncs["bikpin"] = func(text string, u *User, rtm *slack.RTM) Response {
		const N = 400
		return duckduckgoImage("bikini+pineapple", uint(rand.Int31n(N)))
	}
	commandFuncs["squirl"] = func(text string, u *User, rtm *slack.RTM) Response {
		const N = 1000
		return duckduckgoImage("squirrel+images", uint(rand.Int31n(N)))
	}
	commandFuncs["rndimg"] = func(text string, u *User, rtm *slack.RTM) Response {
		const N = 1000
		if text == "" {
			return Response{
				Text: fmt.Sprintf(
					"gets random image from first %d search results", N),
			}
		}
		return duckduckgoImage(text, uint(rand.Int31n(N)))
	}
	commandFuncs["mpoll"] = func(text string, u *User, rtm *slack.RTM) Response {
		return poll(text, true)
	}
	commandFuncs["spoll"] = func(text string, u *User, rtm *slack.RTM) Response {
		return poll(text, false)
	}
	commandFuncs["tr"] = func(text string, u *User, rtm *slack.RTM) Response {
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
	}
	commandFuncs["en"] = func(text string, u *User, rtm *slack.RTM) Response {
		return googleTranslate(text, "en")
	}
	commandFuncs["de"] = func(text string, u *User, rtm *slack.RTM) Response {
		return googleTranslate(text, "de")
	}
	commandFuncs["fact"] = func(text string, u *User, rtm *slack.RTM) Response {
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
	}
	commandFuncs["toon"] = func(text string, u *User, rtm *slack.RTM) Response {
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
	}
	commandFuncs["insult"] = func(text string, u *User, rtm *slack.RTM) Response {
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
	}
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
