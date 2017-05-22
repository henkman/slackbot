package basics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("lvl", func(m adi.Message, rtm *slack.RTM) adi.Response {
		if m.Text == "" {
			return adi.Response{
				Text:   fmt.Sprintf("your level: %d", m.User.Level),
				Charge: true,
			}
		}
		us := adi.GetUserByName(rtm, m.Text)
		if us == nil {
			return adi.Response{
				Text: "user not found",
			}
		}
		up := adi.GetCreateUser(us.ID)
		return adi.Response{
			Text:   fmt.Sprintf("%s level: %d", us.Name, up.Level),
			Charge: true,
		}
	})

	adi.RegisterFunc("setlvl", func(m adi.Message, rtm *slack.RTM) adi.Response {
		if m.Text == "" {
			return adi.Response{
				Text: "set level of user",
			}
		}
		s := strings.Split(m.Text, " ")
		if len(s) != 2 {
			return adi.Response{
				Text: "syntax: setlevel [user] [level]",
			}
		}
		l, err := strconv.ParseUint(s[1], 10, 8)
		if err != nil {
			return adi.Response{
				Text: "syntax: setlevel [user] [level]",
			}
		}
		us := adi.GetUserByName(rtm, s[0])
		if us == nil {
			return adi.Response{
				Text: "user not found",
			}
		}
		up := adi.GetCreateUser(us.ID)
		up.Level = adi.Level(l)
		return adi.Response{
			Text: fmt.Sprintf("%s level is now %d",
				us.Name, up.Level),
			Charge: true,
		}
	})

	adi.RegisterFunc("rqlvl", func(m adi.Message, rtm *slack.RTM) adi.Response {
		if m.Text == "" {
			return adi.Response{
				Text: "find out the required level of a command",
			}
		}
		cmd := adi.GetCommandByName(m.Text)
		if cmd == nil {
			return adi.Response{
				Text: "command not found",
			}
		}
		return adi.Response{
			Text: fmt.Sprintf(
				"%s requires level %d", m.Text, cmd.RequiredLevel),
			Charge: true,
		}
	})

	adi.RegisterFunc("setrqlvl", func(m adi.Message, rtm *slack.RTM) adi.Response {
		if m.Text == "" {
			return adi.Response{
				Text: "set required level for a command",
			}
		}
		s := strings.Split(m.Text, " ")
		if len(s) != 2 {
			return adi.Response{
				Text: "syntax: setrqlvl [command] [level]",
			}
		}
		p, err := strconv.ParseUint(s[1], 10, 8)
		if err != nil {
			return adi.Response{
				Text: "syntax: setrqlvl [command] [level]",
			}
		}
		cmd := adi.GetCommandByName(s[0])
		if cmd == nil {
			return adi.Response{
				Text: "command not found",
			}
		}
		cmd.RequiredLevel = adi.Level(p)
		return adi.Response{
			Text: fmt.Sprintf("%s now requires level %d",
				cmd.Name, cmd.RequiredLevel),
			Charge: true,
		}
	})
}
