package google

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/henkman/google"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

const (
	TLD = "de"
)

var (
	gSess google.Session
)

func init() {

	adi.RegisterFunc("web",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "finds stuff in the internet",
				}
			}
			if !gSess.IsInitialized() {
				if err := gSess.Init(); err != nil {
					log.Println("ERROR:", err.Error())
					return adi.Response{
						Text: "internal error",
					}
				}
			}
			results, err := gSess.Search(TLD, text, "en", false, 0, 5)
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

	adi.RegisterFunc("tr",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
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
				return adi.Response{
					Text: help,
				}
			}
			s := strings.Index(text, " ")
			if s == -1 {
				return adi.Response{
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
					return adi.Response{
						Text: "language not supported",
					}
				}
			}
			t := text[s:]
			return googleTranslate(t, l)
		})

	adi.RegisterFunc("en",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return googleTranslate(text, "en")
		})

	adi.RegisterFunc("de",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return googleTranslate(text, "de")
		})
}

func googleImage(text string, safe bool, typ google.ImageType) adi.Response {
	if text == "" {
		return adi.Response{
			Text: "finds images",
		}
	}
	if !gSess.IsInitialized() {
		if err := gSess.Init(); err != nil {
			log.Println("ERROR:", err.Error())
			return adi.Response{
				Text: "internal error",
			}
		}
	}
	images, err := gSess.Images(TLD, text, "de", safe, typ, 0, 50)
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

func googleTranslate(text, tl string) adi.Response {
	if text == "" {
		return adi.Response{
			Text: "finds images",
		}
	}
	if !gSess.IsInitialized() {
		if err := gSess.Init(); err != nil {
			log.Println("ERROR:", err.Error())
			return adi.Response{
				Text: "internal error",
			}
		}
	}
	lt, err := gSess.Translate(text, "auto", tl)
	if err != nil {
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	return adi.Response{
		Text:   lt.Text,
		Charge: true,
	}
}
