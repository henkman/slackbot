package proxycommands

import (
	"fmt"
	"strings"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

func init() {

	adi.RegisterFunc("setproxy",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "sets a proxy command",
				}
			}
			s := strings.Split(text, " ")
			if len(s) < 2 {
				return adi.Response{
					Text: "syntax: setproxy [name] [cmd]",
				}
			}
			n := s[0]
			t := adi.UrlUnFurl(strings.Join(s[1:], " "))
			c := adi.GetCommandByName(n)
			if c == nil {
				adi.Commands = append(adi.Commands, adi.Command{
					Name:          n,
					Proxy:         t,
					RequiredLevel: adi.DefaultLevel,
					Price:         0,
					Visible:       false,
				})
				adi.ResetCommands()
			} else {
				if c.Func != nil {
					return adi.Response{
						Text: fmt.Sprintf("%s is not a proxy command", s[0]),
					}
				}
				c.Proxy = t
			}
			return adi.Response{
				Text:   fmt.Sprintf("set %s to \"%s\"", n, t),
				Charge: true,
			}
		})

	adi.RegisterFunc("delproxy",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "deletes a proxy command",
				}
			}
			o := -1
			for i, _ := range adi.Commands {
				if adi.Commands[i].Proxy != "" && adi.Commands[i].Name == text {
					o = i
					break
				}
			}
			if o == -1 {
				return adi.Response{
					Text: "command does not exist",
				}
			}
			adi.Commands = append(adi.Commands[:o], adi.Commands[o+1:]...)
			adi.ResetCommands()
			return adi.Response{
				Text:   fmt.Sprintf("%s deleted", text),
				Charge: true,
			}
		})
}
