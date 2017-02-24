package duckduckgo

import (
	"fmt"
	"log"

	"github.com/henkman/duckduckgo"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

var (
	sess duckduckgo.Session
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
			if !sess.IsInitialized() {
				if err := sess.Init(); err != nil {
					log.Println("ERROR:", err)
					return adi.Response{
						Text: "internal error",
					}
				}
			}
			vids, err := sess.Videos(text, 0)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "internal error",
				}
			}
			if len(vids) == 0 {
				return adi.Response{
					Text: "nothing found",
				}
			}
			o := adi.RandUint32(uint32(len(vids)))
			return adi.Response{
				Text:        "https://www.youtube.com/watch?v=" + vids[o].Id,
				Charge:      true,
				UnfurlLinks: true,
			}
		})
}

func duckduckgoImage(query string, offset uint) adi.Response {
	if !sess.IsInitialized() {
		if err := sess.Init(); err != nil {
			log.Println("ERROR:", err)
			return adi.Response{
				Text: "internal error",
			}
		}
	}
	images, err := sess.Images(query, offset)
	if err != nil {
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	if len(images) == 0 {
		return adi.Response{
			Text: "nothing found",
		}
	}
	o := adi.RandUint32(uint32(len(images)))
	return adi.Response{
		Text:        images[o].Url,
		Charge:      true,
		UnfurlLinks: true,
	}
}
