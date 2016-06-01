package basics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("lvl", func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
		if text == "" {
			return adi.Response{
				Text:   fmt.Sprintf("your level: %d", u.Level),
				Charge: true,
			}
		}
		us := adi.GetUserByName(rtm, text)
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

	adi.RegisterFunc("setlvl", func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
		if text == "" {
			return adi.Response{
				Text: "set level of user",
			}
		}
		s := strings.Split(text, " ")
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

	adi.RegisterFunc("rqlvl", func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
		if text == "" {
			return adi.Response{
				Text: "find out the required level of a command",
			}
		}
		cmd := adi.GetCommandByName(text)
		if cmd == nil {
			return adi.Response{
				Text: "command not found",
			}
		}
		return adi.Response{
			Text: fmt.Sprintf(
				"%s requires level %d", text, cmd.RequiredLevel),
			Charge: true,
		}
	})

	adi.RegisterFunc("setrqlvl", func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
		if text == "" {
			return adi.Response{
				Text: "set required level for a command",
			}
		}
		s := strings.Split(text, " ")
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
