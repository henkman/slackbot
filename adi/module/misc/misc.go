package misc

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"sort"
	"strings"
	"time"

	"github.com/alfredxing/calc/compute"
	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
)

var (
	cvm       *otto.Otto
	startTime time.Time
)

type TimeStamp struct {
	Unix   int64
	Unique string
}

func parseTimestamp(s string) (TimeStamp, error) {
	var ts TimeStamp
	o := strings.IndexByte(s, '.')
	if o == -1 {
		return ts, errors.New("not a valid timestamp")
	}
	unix, err := strconv.ParseInt(s[:o], 10, 64)
	if err != nil {
		return ts, errors.New("not a valid timestamp")
	}
	return TimeStamp{
		Unix:   unix,
		Unique: s[o+1:],
	}, nil
}

func init() {

	startTime = time.Now()

	adi.RegisterFunc("uptime",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			return adi.Response{
				Text:   time.Since(startTime).String(),
				Charge: true,
			}
		})

	adi.RegisterFunc("ping",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			ts, err := parseTimestamp(m.Timestamp)
			if err != nil {
				return adi.Response{
					Text: err.Error(),
				}
			}
			mt := time.Unix(ts.Unix, 0)
			return adi.Response{
				Text:   time.Since(mt).String(),
				Charge: true,
			}
		})

	adi.RegisterFunc("cyrill",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			s := strings.TrimSpace(m.Text)
			if s == "" {
				return adi.Response{
					Text: "prints latin script as cyrillic",
				}
			}
			cyrill := []struct {
				Cyrillic string
				Latin    string
			}{
				{"щ", "schtsch"},
				{"ч", "tsch"},
				{"ж", "sch"},
				{"e", "je"},
				{"ё", "jo"},
				{"ю", "ju"},
				{"я", "ja"},
				{"х", "ch"},
				{"к", "ck"},
				{"кс", "x"},
				{"a", "a"},
				{"б", "b"},
				{"B", "w"},
				{"г", "g"},
				{"д", "d"},
				{"x", "h"},
				{"с", "s"},
				{"з", "s"},
				{"и", "i"},
				{"к", "k"},
				{"л", "l"},
				{"м", "m"},
				{"н", "n"},
				{"о", "o"},
				{"п", "p"},
				{"р", "r"},
				{"т", "t"},
				{"у", "u"},
				{"ф", "f"},
				{"ц", "z"},
				{"ы", "y"},
				{"э", "ä"},
				{"э", "v"},
			}
			s = strings.ToLower(s)
			l := len(s)
			res := make([]byte, 0, l)
			o := 0
		next:
			for o < l {
				r := l - o
				for _, p := range cyrill {
					pl := len(p.Latin)
					if pl <= r && strings.HasPrefix(s[o:], p.Latin) {
						res = append(res, []byte(p.Cyrillic)...)
						o += pl
						continue next
					}
				}
				res = append(res, s[o])
				o++
			}
			return adi.Response{
				Text:   string(res),
				Charge: true,
			}
		})

	adi.RegisterFunc("decyrill",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			s := strings.TrimSpace(m.Text)
			if s == "" {
				return adi.Response{
					Text: "prints cyrillic script as latin",
				}
			}

			return adi.Response{
				Text:   "",
				Charge: true,
			}
		})

	adi.RegisterFunc("hidden",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
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

	adi.RegisterFunc("delmsg",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if m.Text == "" {
				return adi.Response{
					Text: "delete message",
				}
			}
			s := strings.Split(m.Text, " ")
			if len(s) != 2 {
				return adi.Response{
					Text: "syntax: delmsg channel timestamp[,timestamp]",
				}
			}
			c := adi.GetChannelByName(rtm, s[0])
			if c == nil {
				return adi.Response{
					Text: "channel not found",
				}
			}
			ts := strings.Split(s[1], ",")
			for _, t := range ts {
				t = strings.TrimSpace(t)
				ts, err := parseTimestamp(m.Timestamp)
				if err != nil {
					return adi.Response{
						Text: err.Error(),
					}
				}
				if _, _, err := rtm.DeleteMessage(c.ID,
					fmt.Sprintf("%d.%s", ts.Unix, ts.Unique)); err != nil {
					log.Println("ERROR:", err)
					return adi.Response{
						Text: "couldn't delete",
					}
				}
			}
			return adi.Response{
				Text:   "",
				Charge: true,
			}
		})

	adi.RegisterFunc("setvis",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if m.Text == "" {
				return adi.Response{
					Text: "set visiblity of a command",
				}
			}
			s := strings.Split(m.Text, " ")
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
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if m.Text == "" {
				return adi.Response{
					Text: "says something",
				}
			}
			return adi.Response{
				Text:   m.Text,
				Charge: true,
			}
		})

	adi.RegisterFunc("id",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if m.Text == "" {
				return adi.Response{
					Text:   fmt.Sprintf("your id: %s", m.User.ID),
					Charge: true,
				}
			}
			us := adi.GetUserByName(rtm, m.Text)
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
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if m.Text == "" {
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
			res, err := compute.Evaluate(m.Text)
			if err != nil {
				log.Println("ERROR:", err)
				return adi.Response{
					Text: "Error:" + err.Error(),
				}
			}
			return adi.Response{
				Text:   fmt.Sprintf("%s=%g", m.Text, res),
				Charge: true,
			}
		})

	adi.RegisterFunc("coin",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
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
		func(m adi.Message, rtm *slack.RTM) (r adi.Response) {
			if m.Text == "" {
				return adi.Response{
					Text: "interactive javascript console. type reload to reload the VM",
				}
			}
			if m.Text == "reload" {
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
			val, err := cvm.Run(m.Text)
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
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if m.Text == "" {
				return adi.Response{
					Text: "randomly prints one of the comma separated texts given",
				}
			}
			c := strings.Split(m.Text, ",")
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
