package google

import (
	"bytes"
	"fmt"
	"log"
	"net/url"

	"github.com/henkman/google"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

const (
	TLD = "de"
)

var (
	gclient     google.Client
	initialized bool
)

func init() {

	adi.RegisterFunc("web",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "finds stuff in the internet",
				}
			}
			if !initialized {
				if err := gclient.Init(); err != nil {
					log.Println("ERROR:", err.Error())
					return adi.Response{
						Text: "internal error",
					}
				}
				initialized = true
			}
			results, err := gclient.Search(TLD, text, "en", false, 0, 5)
			if err != nil {
				log.Println("ERROR:", err.Error())
				return adi.Response{
					Text: "internal error",
				}
			}
			if len(results) == 0 {
				return adi.Response{
					Text: "nothing found",
				}
			}
			buf := bytes.NewBufferString("")
			for _, res := range results {
				fmt.Fprintf(buf, "%s %s\n", res.URL, res.Content)
			}
			return adi.Response{
				Text:   buf.String(),
				Charge: true,
			}
		})

	adi.RegisterFunc("img",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return googleImage(text, true, google.ImageType_Any)
		})

	adi.RegisterFunc("nsfwimg",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return googleImage(text, false, google.ImageType_Any)
		})

	adi.RegisterFunc("gif",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return googleImage(text, true, google.ImageType_Animated)
		})

	adi.RegisterFunc("nsfwgif",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return googleImage(text, false, google.ImageType_Animated)
		})
}

func googleImage(text string, safe bool, typ google.ImageType) adi.Response {
	if text == "" {
		return adi.Response{
			Text: "finds images",
		}
	}
	if !initialized {
		if err := gclient.Init(); err != nil {
			log.Println("ERROR:", err.Error())
			return adi.Response{
				Text: "internal error",
			}
		}
		initialized = true
	}
	images, err := gclient.Images(TLD, text, "de", safe, typ, 0, 50)
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
	r := adi.RandUint32(uint32(len(images)))
	u, err := url.QueryUnescape(images[r].URL)
	if err != nil {
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	return adi.Response{
		Text:   u,
		Charge: true,
	}
}
