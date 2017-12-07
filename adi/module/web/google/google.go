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

	adi.RegisterFunc("gl",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleSearch(m.Text, true)
		})

	adi.RegisterFunc("glnsfw",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleSearch(m.Text, false)
		})

	adi.RegisterFunc("glimg",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleImage(m.Text, true, google.ImageType_Any)
		})

	adi.RegisterFunc("glimgnsfw",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleImage(m.Text, false, google.ImageType_Any)
		})

	adi.RegisterFunc("glgif",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleImage(m.Text, true, google.ImageType_Animated)
		})

	adi.RegisterFunc("glgifnsfw",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleImage(m.Text, false, google.ImageType_Animated)
		})

	adi.RegisterFunc("tr",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
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
			if m.Text == "" {
				return adi.Response{
					Text: help,
				}
			}
			s := strings.Index(m.Text, " ")
			if s == -1 {
				return adi.Response{
					Text: help,
				}
			}
			l := m.Text[:s]
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
			t := m.Text[s:]
			return googleTranslate(t, l)
		})

	adi.RegisterFunc("en",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleTranslate(m.Text, "en")
		})

	adi.RegisterFunc("de",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return googleTranslate(m.Text, "de")
		})
}

func googleSearch(text string, safe bool) adi.Response {
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
	text = adi.UrlUnFurl(text)
	results, err := gSess.Search(TLD, text, "en", safe, 0, 5)
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
	text = adi.UrlUnFurl(text)
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
		Text:        u,
		Charge:      true,
		UnfurlLinks: true,
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
