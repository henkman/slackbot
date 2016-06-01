package duckduckgo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("bikpin",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			const N = 400
			return duckduckgoImage("bikini+pineapple", uint(adi.RandUint32(N)))
		})

	adi.RegisterFunc("squirl",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			const N = 1000
			return duckduckgoImage("squirrel+images", uint(adi.RandUint32(N)))
		})

	adi.RegisterFunc("rndimg",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			const N = 1000
			if text == "" {
				return adi.Response{
					Text: fmt.Sprintf(
						"gets random image from first %d search results", N),
				}
			}
			return duckduckgoImage(text, uint(adi.RandUint32(N)))
		})

	adi.RegisterFunc("vid",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
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
					return adi.Response{
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
				return adi.Response{
					Text: "internal error",
				}
			}
			r.Body.Close()
			if len(ytr.Results) == 0 {
				return adi.Response{
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
				return adi.Response{
					Text: "nothing found",
				}
			}
			o := adi.RandUint32(uint32(len(ytr.Results)))
			return adi.Response{
				Text:   "https://www.youtube.com/watch?v=" + ids[o],
				Charge: true,
			}
		})
}

func duckduckgoImage(query string, offset uint) adi.Response {
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
			return adi.Response{
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
		return adi.Response{
			Text: "nothing found",
		}
	}
	r.Body.Close()
	if len(ytr.Results) == 0 {
		return adi.Response{
			Text: "nothing found",
		}
	}
	o := adi.RandUint32(uint32(len(ytr.Results)))
	return adi.Response{
		Text:   ytr.Results[o].Image,
		Charge: true,
	}
}
