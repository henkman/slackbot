package poll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("mpoll",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return poll(text, true)
		})

	adi.RegisterFunc("spoll",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			return poll(text, false)
		})
}

func poll(text string, multi bool) adi.Response {
	if text == "" {
		return adi.Response{
			Text: `creates a poll
Example: poll animal?, dog, cat, hamster
-> creates a poll with title animal? and the three animals as choices`,
		}
	}
	s := strings.Split(text, ",")
	if len(s) < 3 {
		return adi.Response{
			Text: "needs one question and at least 2 options",
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
		return adi.Response{
			Text: "internal error",
		}
	}
	r, err := http.Post("https://www.strawpoll.me/api/v2/polls",
		"application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	var pres struct {
		ID uint64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&pres); err != nil {
		r.Body.Close()
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	r.Body.Close()
	p := fmt.Sprintf("http://www.strawpoll.me/%d", pres.ID)
	log.Println("new poll:", p)
	return adi.Response{
		Text:   p,
		Charge: true,
	}
}
