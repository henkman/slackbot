package main

import (
	"bytes"
	"fmt"
	"log"
	"math"

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
func (a UsersByRank) Less(i, j int) bool { return a[i].Points > a[j].Points }

func init() {
	commandFuncs["lottery"] = func(text string, u *User, rtm *slack.RTM) Response {
		lot := &bank.Lottery
		if text == "" {
			return Response{
				Text: fmt.Sprintf(
					"tickets[price:%d, sold:%d], drawing:%s, pot:%d | try 'lottery help' for help",
					lot.TicketPrice,
					lot.TicketsSold,
					lot.LastDraw.Add(lot.DrawEvery).UTC().Format("02.Jan 15:04 MST"),
					lot.Pot,
				),
			}
		}
		if text == "help" {
			return Response{
				Text: `the lottery is drawn periodically. users can buy multiple tickets.
one of the sold tickets is chosen as winner and gets the whole pot.
if a drawing comes up and only one user bought tickets:
	- bought ticket(s) stay in the game
	- bank pays a small sum into the pot if it has the cash
use 'lottery [tickets|all]' to buy tickets, 'lottery info' to get infos`,
			}
		}
		if text == "info" {
			if lot.TicketsSold == 0 {
				return Response{
					Text: "no one has bought a ticket",
				}
			}
			var t string
			ts, ok := lot.Tickets[u.ID]
			if ok {
				t = fmt.Sprintf(
					"you have %d tickets. %d other users bought %d tickets",
					ts, len(lot.Tickets)-1, lot.TicketsSold-ts)
			} else {
				t = fmt.Sprintf(
					"you did not buy tickets. %d other users bought %d tickets",
					len(lot.Tickets), lot.TicketsSold)
			}
			return Response{
				Text: t,
			}
		}
		var src Account = &u.Points
		var n uint64
		{
			if text == "all" {
				if src.Balance() < lot.TicketPrice {
					return Response{
						Text: "you do not have enough points.",
					}
				}
				n = uint64(src.Balance() / lot.TicketPrice)
			} else {
				t, err := strconv.ParseUint(text, 10, 64)
				if err != nil {
					return Response{
						Text: "syntax: lottery [tickets|all]",
					}
				}
				if t == 0 {
					return Response{
						Text: "needs to be at least 1",
					}
				}
				if mulOverflows(uint64(lot.TicketPrice), t) ||
					lot.TicketsSold > (math.MaxUint64-t) {
					return Response{
						Text: "can't buy that much tickets",
					}
				}
				n = t
			}
		}
		p := lot.TicketPrice * Points(n)
		if p > src.Balance() {
			return Response{
				Text: "you do not have enough points.",
			}
		}
		ts, ok := lot.Tickets[u.ID]
		if ok {
			if ts > (math.MaxUint64 - n) {
				return Response{
					Text: "can't buy that much tickets",
				}
			}
			lot.Tickets[u.ID] += n
		} else {
			lot.Tickets[u.ID] = n
		}
		lot.TicketsSold += n
		src.Sub(p)
		lot.Pot.Add(p)
		return Response{
			Text: fmt.Sprintf("you bought %d tickets for %d. your points:%d. pot: %d",
				n, p, src.Balance(), lot.Pot.Balance(),
			),
			Charge: true,
		}
	}
	commandFuncs["rank"] = func(text string, u *User, rtm *slack.RTM) Response {
		us := make([]User, len(users))
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
					break
				}
			}
		}
		return Response{
			Text:   s.String(),
			Charge: true,
		}
	}
	commandFuncs["trpts"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "transfer points",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 3 {
			return Response{
				Text: "syntax: trpts [src] [dst] [points|all]",
			}
		}
		src := getAccountByName(rtm, s[0])
		if src == nil {
			return Response{
				Text: fmt.Sprintf("user %s not found", s[0]),
			}
		}
		n, err := parsePoints(src, s[0], s[2])
		if err != "" {
			return Response{
				Text: err,
			}
		}
		dst := getAccountByName(rtm, s[1])
		if dst == nil {
			return Response{
				Text: fmt.Sprintf("user %s not found", s[1]),
			}
		}
		if src == dst {
			return Response{
				Text: "source and destination can not be the same",
			}
		}
		src.Sub(n)
		dst.Add(n)
		return Response{
			Text: fmt.Sprintf("%s points are now %d. %s points are now %d",
				s[0], src.Balance(), s[1], dst.Balance()),
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
				Text: "syntax: setlevel [user] [level]",
			}
		}
		l, err := strconv.ParseUint(s[1], 10, 8)
		if err != nil {
			return Response{
				Text: "syntax: setlevel [user] [level]",
			}
		}
		us := getUserByName(rtm, s[0])
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
		cmd.Price = Points(p)
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
				Text: "give points to user",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: givepts [user] [points|all]",
			}
		}
		var src, dst Account
		src = &u.Points
		n, err := parsePoints(src, "", s[1])
		if err != "" {
			return Response{
				Text: err,
			}
		}
		dst = getAccountByName(rtm, s[0])
		if dst == nil {
			return Response{
				Text: "user not found",
			}
		}
		if src == dst {
			return Response{
				Text: "can't give points to yourself",
			}
		}
		src.Sub(n)
		dst.Add(n)
		return Response{
			Text: fmt.Sprintf("%s points %d. your points: %d",
				s[0], dst.Balance(), src.Balance()),
			Charge: true,
		}
	}
	commandFuncs["duel"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text: "challenge somebody to get their points",
			}
		}
		s := strings.Split(text, " ")
		if len(s) != 2 {
			return Response{
				Text: "syntax: duel [user] [points|all]",
			}
		}
		if s[0] == "pot" {
			return Response{
				Text: fmt.Sprintf("can't duel %s", s[0]),
			}
		}
		var src, dst Account
		src = &u.Points
		dst = getAccountByName(rtm, s[0])
		if dst == nil {
			return Response{
				Text: "user not found",
			}
		}
		if src == dst {
			return Response{
				Text: "can't duel yourself",
			}
		}
		if dst.Balance() == 0 {
			return Response{
				Text: fmt.Sprintf("%s has no points", s[0]),
			}
		}
		var n Points
		if s[1] == "all" {
			if src.Balance() == 0 {
				return Response{
					Text: "you have no points",
				}
			}
			if src.Balance() > dst.Balance() {
				n = dst.Balance()
			} else {
				n = src.Balance()
			}
		} else {
			t, err := strconv.ParseUint(s[1], 10, 64)
			if err != nil {
				return Response{
					Text: "syntax: duel [user] [points|all]",
				}
			}
			if t == 0 {
				return Response{
					Text: "Must be more than 0 points",
				}
			}
			if Points(t) > src.Balance() {
				return Response{
					Text: fmt.Sprintf(
						"not enough points. your points: %d",
						src.Balance()),
				}
			}
			if Points(t) > dst.Balance() {
				return Response{
					Text: fmt.Sprintf(
						"%s does not have enough points. %s points: %d",
						s[0], s[0], dst.Balance()),
				}
			}
			n = Points(t)
		}
		var t string
		if randBool() {
			src.Add(n)
			dst.Sub(n)
			t = fmt.Sprintf("you took %d points. your points: %d. %s points: %d",
				n, src.Balance(), s[0], dst.Balance())
		} else {
			src.Sub(n)
			dst.Add(n)
			t = fmt.Sprintf("you lost %d points. your points: %d. %s points: %d",
				n, src.Balance(), s[0], dst.Balance())
		}
		return Response{
			Text:   t,
			Charge: true,
		}
	}
	commandFuncs["pts"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text:   fmt.Sprintf("your points: %d", u.Points),
				Charge: true,
			}
		}
		src := getAccountByName(rtm, text)
		if src == nil {
			return Response{
				Text: "user not found",
			}
		}
		return Response{
			Text:   fmt.Sprintf("%s points: %d", text, src.Balance()),
			Charge: true,
		}
	}
	commandFuncs["lvl"] = func(text string, u *User, rtm *slack.RTM) Response {
		if text == "" {
			return Response{
				Text:   fmt.Sprintf("your level: %d", u.Level),
				Charge: true,
			}
		}
		us := getUserByName(rtm, text)
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
				Text:   fmt.Sprintf("your id: %s", u.ID),
				Charge: true,
			}
		}
		us := getUserByName(rtm, text)
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
		var t string
		if randBool() {
			t = "heads"
		} else {
			t = "tails"
		}
		return Response{
			Text:   t,
			Charge: true,
		}
	}
	commandFuncs["js"] = func(text string, u *User, rtm *slack.RTM) (r Response) {
		if text == "" {
			return Response{
				Text: "interactive javascript console. type reload to reload the VM",
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
		t := c[randUint32(uint32(len(c)))]
		return Response{
			Text:   strings.TrimSpace(t),
			Charge: true,
		}
	}
}

func getUserByName(rtm *slack.RTM, name string) *slack.User {
	us, err := rtm.GetUsers()
	if err != nil {
		log.Println("ERROR:", err.Error())
		return nil
	}
	for _, o := range us {
		if o.Name == name {
			return &o
		}
	}
	return nil
}

func getAccountByName(rtm *slack.RTM, name string) Account {
	if name == "bank" {
		return &bank.Points
	}
	if name == "pot" {
		return &bank.Lottery.Pot
	}
	us := getUserByName(rtm, name)
	if us == nil {
		return nil
	}
	return &getCreateUser(us.ID).Points
}

func parsePoints(src Account, name, text string) (Points, string) {
	var n Points
	if text == "all" {
		if src.Balance() == 0 {
			if len(name) == 0 {
				return 0, "you have no points"
			} else {
				return 0, fmt.Sprintf("%s got no points", name)
			}
		}
		n = src.Balance()
	} else {
		t, err := strconv.ParseUint(text, 10, 64)
		if err != nil || t == 0 {
			return 0, "points have to be positive"
		}
		if src.Balance() < Points(t) {
			if len(name) == 0 {
				return 0, fmt.Sprintf(
					"you do not have enough points. you have %d",
					src.Balance())
			} else {
				return 0, fmt.Sprintf(
					"%s does not have enough points. %s has %d",
					name, name, src.Balance())
			}
		}
		n = Points(t)
	}
	return n, ""
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
