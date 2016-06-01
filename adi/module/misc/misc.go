package admin

import (
	"fmt"
	"log"

	"sort"
	"strings"
	"time"

	"github.com/alfredxing/calc/compute"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
)

var (
	cvm *otto.Otto
)

func init() {

	adi.RegisterFunc("hidden",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			hidden := make([]string, 0, len(adi.Commands))
			for _, c := range adi.Commands {
				if !c.Visible {
					hidden = append(hidden, c.Name)
				}
			}
			sort.Sort(sort.StringSlice(hidden))
			return adi.Response{
				Text:   strings.Join(hidden, ", "),
				Charge: true,
			}
		})

	adi.RegisterFunc("setvis",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "set visiblity of a command",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return adi.Response{
					Text: "syntax: setvis [command] [visible|hidden]",
				}
			}
			cmd := adi.GetCommandByName(s[0])
			if cmd == nil {
				return adi.Response{
					Text: "command not found",
				}
			}
			cmd.Visible = s[1] == "visible"
			adi.ResetCommands()
			return adi.Response{
				Text:   fmt.Sprintf("%s now costs %d", cmd.Name, cmd.Price),
				Charge: true,
			}
		})

	adi.RegisterFunc("say",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "says something",
				}
			}
			return adi.Response{
				Text:   text,
				Charge: true,
			}
		})

	adi.RegisterFunc("id",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text:   fmt.Sprintf("your id: %s", u.ID),
					Charge: true,
				}
			}
			us := adi.GetUserByName(rtm, text)
			if us == nil {
				return adi.Response{
					Text: "user not found",
				}
			}
			return adi.Response{
				Text:   fmt.Sprintf("%s id: %s", us.Name, us.ID),
				Charge: true,
			}
		})

	adi.RegisterFunc("calc",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: `a calculator
	Usage:
	  Operators: +, -, *, /, ^, %%
	  Functions: sin, cos, tan, cot, sec, csc,
	             asin, acos, atan, acot, asec,
	             acsc, sqrt, log, lg, ln, abs
	  Constants: e, pi, π`,
				}
			}
			res, err := compute.Evaluate(text)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "Error:" + err.Error(),
				}
			}
			return adi.Response{
				Text:   fmt.Sprintf("%s=%g", text, res),
				Charge: true,
			}
		})

	adi.RegisterFunc("coin",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			var t string
			if adi.RandBool() {
				t = "heads"
			} else {
				t = "tails"
			}
			return adi.Response{
				Text:   t,
				Charge: true,
			}
		})

	adi.RegisterFunc("js",
		func(text string, u *adi.User, rtm *slack.RTM) (r adi.Response) {
			if text == "" {
				return adi.Response{
					Text: "interactive javascript console. type reload to reload the VM",
				}
			}
			if text == "reload" {
				cvm = otto.New()
				cvm.Set("console", otto.UndefinedValue())
				return adi.Response{
					Text: "VM reloaded",
				}
			}
			if cvm == nil {
				cvm = otto.New()
				cvm.Set("console", otto.UndefinedValue())
			}
			cvm.Interrupt = make(chan func(), 1)
			defer func() {
				if timeout := recover(); timeout != nil {
					r = adi.Response{
						Text: "Code took too long",
					}
				}
			}()
			go func() {
				time.Sleep(time.Second)
				cvm.Interrupt <- func() {
					panic("timeout")
				}
			}()
			val, err := cvm.Run(text)
			if err != nil {
				r = adi.Response{
					Text: err.Error(),
				}
			} else {
				r = adi.Response{
					Text:   val.String(),
					Charge: true,
				}
			}
			return
		})

	adi.RegisterFunc("rnd",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "randomly prints one of the comma separated texts given",
				}
			}
			c := strings.Split(text, ",")
			if len(c) == 1 {
				return adi.Response{
					Text:   strings.TrimSpace(c[0]),
					Charge: true,
				}
			}
			t := c[adi.RandUint32(uint32(len(c)))]
			return adi.Response{
				Text:   strings.TrimSpace(t),
				Charge: true,
			}
		})
}
