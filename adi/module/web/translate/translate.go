package translate

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

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

func googleTranslate(text, tl string) adi.Response {
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
		return adi.Response{
			Text: "internal error",
		}
	}
	tj := make([]string, 0, 2)
	if err := json.NewDecoder(r.Body).Decode(&tj); err != nil {
		r.Body.Close()
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	r.Body.Close()
	if len(tj) != 2 {
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	t, l := tj[0], tj[1]
	return adi.Response{
		Text:   fmt.Sprintf("%s: %s", l, t),
		Charge: true,
	}
}
