package web

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

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
