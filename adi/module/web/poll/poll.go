package poll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("pollmul",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return poll(m.Text, true)
		})

	adi.RegisterFunc("poll",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return poll(m.Text, false)
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
	res, err := adi.HttpPostWithTimeout(
		"https://www.strawpoll.me/api/v2/polls",
		"application/json", bytes.NewBuffer(data), time.Second*10)
	if err != nil {
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	var pres struct {
		ID uint64 `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&pres); err != nil {
		res.Body.Close()
		log.Println("ERROR:", err)
		return adi.Response{
			Text: "internal error",
		}
	}
	res.Body.Close()
	p := fmt.Sprintf("http://www.strawpoll.me/%d", pres.ID)
	log.Println("new poll:", p)
	return adi.Response{
		Text:        p,
		Charge:      true,
		UnfurlLinks: true,
	}
}
