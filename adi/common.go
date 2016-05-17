package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alfredxing/calc/compute"
	"github.com/nlopes/slack"
	"github.com/robertkrimen/otto"
)

type UsersByRank []User

func (a UsersByRank) Len() int           { return len(a) }
func (a UsersByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UsersByRank) Less(i, j int) bool { return a[i].Points < a[j].Points }

func init() {
	commandFuncs["rank"] = func(text string, u *User, rtm *slack.RTM) Response {
		us := make([]User, 0, len(users))
		for i, o := range users {
			us[i] = o
		}
		sort.Sort(UsersByRank(us))

		sus, err := rtm.GetUsers()
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		s := bytes.NewBufferString("")
		for i, o := range us {
			for _, su := range sus {
				if su.ID == o.ID {
					fmt.Fprintf(s, "%d. %s (%d)\n",
						i+1, su.Name, o.Points)
				}
			}
		}
		return Response{
			Text:   s.String(),
			Charge: true,
		}
	}
	commandFuncs["setpts"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "set points of user",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: setpoints [username] [points]",
			}
		}
		p, err := strconv.ParseUint(s[1], 10, 64)
		if err != nil {
			return Response{
				Text: "syntax: setpoints [username] [points]",
			}
		}
		us, err := getUserByName(rtm, s[0])
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		if us == nil {
			return Response{
				Text: "user not found",
			}
		}
		up := getCreateUser(us.ID)
		up.Points = p
		return Response{
			Text: fmt.Sprintf("%s points are now %d",
				us.Name, up.Points),
			Charge: true,
		}
	}
	commandFuncs["setlvl"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "set level of user",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: setlevel [username] [level]",
			}
		}
		l, err := strconv.ParseUint(s[1], 10, 8)
		if err != nil {
			return Response{
				Text: "syntax: setlevel [username] [level]",
			}
		}
		us, err := getUserByName(rtm, s[0])
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		if us == nil {
			return Response{
				Text: "user not found",
			}
		}
		up := getCreateUser(us.ID)
		up.Level = Level(l)
		return Response{
			Text: fmt.Sprintf("%s level is now %d",
				us.Name, up.Level),
			Charge: true,
		}
	}
	commandFuncs["cost"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "find out the price of a command",
			}
		}
		cmd := getCommandByName(text)
		if cmd == nil {
			return Response{
				Text: "command not found",
			}
		}
		return Response{
			Text:   fmt.Sprintf("%s costs %d", text, cmd.Price),
			Charge: true,
		}
	}
	commandFuncs["setprc"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "set price of a command",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: setprc [command] [price]",
			}
		}
		p, err := strconv.ParseUint(s[1], 10, 64)
		if err != nil {
			return Response{
				Text: "syntax: setprc [command] [price]",
			}
		}
		cmd := getCommandByName(s[0])
		if cmd == nil {
			return Response{
				Text: "command not found",
			}
		}
		cmd.Price = p
		return Response{
			Text:   fmt.Sprintf("%s now costs %d", cmd.Name, cmd.Price),
			Charge: true,
		}
	}
	commandFuncs["setrqlvl"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "set required level for a command",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: setrqlvl [command] [level]",
			}
		}
		p, err := strconv.ParseUint(s[1], 10, 8)
		if err != nil {
			return Response{
				Text: "syntax: setrqlvl [command] [level]",
			}
		}
		cmd := getCommandByName(s[0])
		if cmd == nil {
			return Response{
				Text: "command not found",
			}
		}
		cmd.RequiredLevel = Level(p)
		return Response{
			Text: fmt.Sprintf("%s now requires level %d",
				cmd.Name, cmd.RequiredLevel),
			Charge: true,
		}
	}
	commandFuncs["givepts"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "give points to username",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: givepoints [username] [points|all]",
			}
		}
		un := s[0]
		var n uint64
		if s[1] == "all" {
			n = u.Points
		} else {
			t, err := strconv.ParseUint(s[1], 10, 64)
			if err != nil {
				return Response{
					Text: "syntax: givepoints [username] [points|all]",
				}
			}
			if t > u.Points {
				return Response{
					Text: fmt.Sprintf(
						"not enough points. your points: %d",
						u.Points),
				}
			}
			n = t
		}
		if n == 0 {
			return Response{
				Text: "Must be more than 0 points",
			}
		}
		us, err := getUserByName(rtm, un)
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		if us == nil {
			return Response{
				Text: "user not found",
			}
		}
		if us.ID == u.ID {
			return Response{
				Text: "can't give points to yourself",
			}
		}
		up := getCreateUser(us.ID)
		up.Add(n)
		u.Sub(n)
		return Response{
			Text: fmt.Sprintf("%s points %d. your points: %d",
				us.Name, up.Points, u.Points),
			Charge: true,
		}
	}
	commandFuncs["bet"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "bet points to double them",
			}
		}
		var n uint64
		if text == "all" {
			n = u.Points
		} else {
			t, err := strconv.ParseUint(text, 10, 64)
			if err != nil {
				return Response{
					Text: "syntax: roulette [points|all]",
				}
			}
			if t > u.Points {
				return Response{
					Text: fmt.Sprintf(
						"not enough points. your points: %d",
						u.Points),
				}
			}
			n = t
		}
		if n == 0 {
			return Response{
				Text: "Must be more than 0 points",
			}
		}
		var t string
		if rand.Int31n(2) == 1 {
			n *= 2
			u.Add(n)
			t = fmt.Sprintf("You won %d points. New points %d",
				n, u.Points)
		} else {
			u.Sub(n)
			t = fmt.Sprintf("You lost %d points. New points %d",
				n, u.Points)
		}
		return Response{
			Text:   t,
			Charge: true,
		}
	}
	commandFuncs["pts"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: fmt.Sprintf("your points: %d", u.Points),
			}
		}
		us, err := getUserByName(rtm, text)
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		if us == nil {
			return Response{
				Text: "user not found",
			}
		}
		up := getCreateUser(us.ID)
		return Response{
			Text:   fmt.Sprintf("%s points: %d", us.Name, up.Points),
			Charge: true,
		}
	}
	commandFuncs["lvl"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: fmt.Sprintf("your level: %d", u.Level),
			}
		}
		us, err := getUserByName(rtm, text)
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		if us == nil {
			return Response{
				Text: "user not found",
			}
		}
		up := getCreateUser(us.ID)
		return Response{
			Text:   fmt.Sprintf("%s level: %d", us.Name, up.Level),
			Charge: true,
		}
	}
	commandFuncs["id"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: fmt.Sprintf("your id: %s", u.ID),
			}
		}
		us, err := getUserByName(rtm, text)
		if err != nil {
			log.Println("ERROR:", err.Error())
			return Response{
				Text: "internal error",
			}
		}
		if us == nil {
			return Response{
				Text: "user not found",
			}
		}
		return Response{
			Text:   fmt.Sprintf("%s id: %s", us.Name, us.ID),
			Charge: true,
		}
	}
	commandFuncs["calc"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: `A Calculator
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
			return Response{
				Text: "Error:" + err.Error(),
			}
		}
		return Response{
			Text:   fmt.Sprintf("%s=%g", text, res),
			Charge: true,
		}
	}
	commandFuncs["coin"] = func(text string, u *User, rtm *slack.RTM) Response {
		return Response{
			Text: []string{
				"heads",
				"tails",
			}[rand.Int31n(2)],
			Charge: true,
		}
	}
	commandFuncs["js"] = func(text string, u *User, rtm *slack.RTM) (r Response) {
		if text == "" {
			return Response{
				Text: "interactive javascript console\nType reload to reload the VM",
			}
		}
		if text == "reload" {
			cvm = otto.New()
			cvm.Set("console", otto.UndefinedValue())
			return Response{
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
				r = Response{
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
			r = Response{
				Text: err.Error(),
			}
		} else {
			r = Response{
				Text:   val.String(),
				Charge: true,
			}
		}
		return
	}
	commandFuncs["rnd"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "randomly prints one of the comma separated texts given",
			}
		}
		c := strings.Split(text, ",")
		if len(c) == 1 {
			return Response{
				Text:   strings.TrimSpace(c[0]),
				Charge: true,
			}
		}
		t := c[rand.Int31n(int32(len(c)))]
		return Response{
			Text:   strings.TrimSpace(t),
			Charge: true,
		}
	}
}

func getUserByName(rtm *slack.RTM, name string) (*slack.User, error) {
	us, err := rtm.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, o := range us {
		if o.Name == name {
			return &o, nil
		}
	}
	return nil, nil
}

func getCreateUser(id string) *User {
	for i, _ := range users {
		if users[i].ID == id {
			return &users[i]
		}
	}
	user := User{ID: id, Level: defaultLevel, Points: 0}
	users = append(users, user)
	return &users[len(users)-1]
}
