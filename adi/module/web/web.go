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

	"github.com/PuerkitoBio/goquery"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("synonym",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			q := strings.TrimSpace(text)
			if q == "" {
				return adi.Response{
					Text: "finds synonyms",
				}
			}
			q = strings.ToLower(q)
			r, err := http.Get(fmt.Sprintf(
				"https://www.openthesaurus.de/synonyme/search?q=%s&format=application/json",
				url.QueryEscape(q)))
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
			if err := json.NewDecoder(r.Body).Decode(&synsets); err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
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
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			var r *http.Response
			{
				var err error
				r, err = http.Get(fmt.Sprintf(
					"https://api.dubtrack.fm/room/%s",
					adi.DubtrackRoom))
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
			if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
				r.Body.Close()
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			r.Body.Close()
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
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			doc, err := goquery.NewDocument("http://randomfunfacts.com/")
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

	adi.RegisterFunc("toon",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			doc, err := goquery.NewDocument("http://www.veryfunnycartoons.com/")
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
				Text:   img,
				Charge: true,
			}
		})

	adi.RegisterFunc("insult",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			doc, err := goquery.NewDocument("http://www.randominsults.net/")
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
