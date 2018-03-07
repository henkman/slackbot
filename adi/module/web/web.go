package web

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("synonym",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			q := strings.TrimSpace(m.Text)
			if q == "" {
				return adi.Response{
					Text: "finds synonyms",
				}
			}
			q = strings.ToLower(q)
			res, err := adi.HttpGetWithTimeout(fmt.Sprintf(
				"https://www.openthesaurus.de/synonyme/search?q=%s&format=application/json",
				url.QueryEscape(q)),
				time.Second*10)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			var synsets struct {
				Synsets []struct {
					Terms []struct {
						Term string `json:"term"`
					} `json:"terms"`
				} `json:"synsets"`
			}
			if err := json.NewDecoder(res.Body).Decode(&synsets); err != nil {
				res.Body.Close()
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			res.Body.Close()
			syns := make([]string, 0, 10)
			for _, ss := range synsets.Synsets {
				for _, t := range ss.Terms {
					term := strings.TrimSpace(t.Term)
					if strings.ToLower(term) != q {
						syns = append(syns, term)
					}
				}
			}
			if len(syns) == 0 {
				return adi.Response{
					Text: "no synonyms found",
				}
			}
			sort.Strings(syns)
			n := adi.Uniq(sort.StringSlice(syns))
			return adi.Response{
				Text:   strings.Join(syns[:n], ", "),
				Charge: true,
			}
		})

	adi.RegisterFunc("song",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			var res *http.Response
			{
				var err error
				res, err = adi.HttpGetWithTimeout(
					fmt.Sprintf(
						"https://api.dubtrack.fm/room/%s",
						adi.DubtrackRoom),
					time.Second*10)
				if err != nil {
					log.Println("ERROR:", err)
					return adi.Response{
						Text: "internal error",
					}
				}
			}
			var room struct {
				Data struct {
					ActiveUsers int `json:"activeUsers"`
					CurrentSong *struct {
						Name string `json:"name"`
					} `json:"currentSong"`
				} `json:"data"`
			}
			if err := json.NewDecoder(res.Body).Decode(&room); err != nil {
				res.Body.Close()
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			res.Body.Close()
			var t string
			d := room.Data
			if d.CurrentSong == nil {
				t = fmt.Sprintf("Currently playing nothing. %d are listening",
					d.ActiveUsers)
			} else {
				n := html.UnescapeString(d.CurrentSong.Name)
				t = fmt.Sprintf("Currently playing \"%s\". %d are listening",
					n, d.ActiveUsers)
			}
			return adi.Response{
				Text:   t,
				Charge: true,
			}
		})

	adi.RegisterFunc("fact",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			res, err := adi.HttpGetWithTimeout(
				"http://randomfunfacts.com/",
				time.Second*10)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			doc, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			return adi.Response{
				Text:        doc.Find("center i").Text(),
				Charge:      true,
				UnfurlLinks: true,
			}
		})

	adi.RegisterFunc("toon",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			res, err := adi.HttpGetWithTimeout(
				"http://www.veryfunnycartoons.com/",
				time.Second*10)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			doc, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			img, ok := doc.Find("center i img").Attr("src")
			if !ok {
				log.Println("ERROR: cartoon img src not found")
				return adi.Response{
					Text: "internal error",
				}
			}
			return adi.Response{
				Text:        img,
				Charge:      true,
				UnfurlLinks: true,
			}
		})

	adi.RegisterFunc("insult",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			res, err := adi.HttpGetWithTimeout("http://www.randominsults.net/",
				time.Second*10)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			doc, err := goquery.NewDocumentFromResponse(res)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			return adi.Response{
				Text:   doc.Find("center i").Text(),
				Charge: true,
			}
		})
}
