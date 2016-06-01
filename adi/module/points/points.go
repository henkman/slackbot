package points

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/henkman/slackbot/adi"
	"github.com/nlopes/slack"
)

type UsersByRank []adi.User

func (a UsersByRank) Len() int           { return len(a) }
func (a UsersByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UsersByRank) Less(i, j int) bool { return a[i].Points > a[j].Points }

func init() {

	adi.RegisterFunc("rank",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			us := make([]adi.User, len(adi.Users))
			copy(us, adi.Users)
			sort.Sort(UsersByRank(us))
			sus, err := rtm.GetUsers()
			if err != nil {
				log.Println("ERROR:", err.Error())
				return adi.Response{
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
			return adi.Response{
				Text:   s.String(),
				Charge: true,
			}
		})

	adi.RegisterFunc("setprc",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "set price of a command",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return adi.Response{
					Text: "syntax: setprc [command] [price]",
				}
			}
			p, err := strconv.ParseUint(s[1], 10, 64)
			if err != nil {
				return adi.Response{
					Text: "syntax: setprc [command] [price]",
				}
			}
			cmd := adi.GetCommandByName(s[0])
			if cmd == nil {
				return adi.Response{
					Text: "command not found",
				}
			}
			cmd.Price = adi.Points(p)
			return adi.Response{
				Text:   fmt.Sprintf("%s now costs %d", cmd.Name, cmd.Price),
				Charge: true,
			}
		})

	adi.RegisterFunc("givepts",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "give points to user",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return adi.Response{
					Text: "syntax: givepts [user] [points|all]",
				}
			}
			var src, dst adi.Account
			src = &u.Points
			n, err := adi.ParsePoints(src, "", s[1])
			if err != "" {
				return adi.Response{
					Text: err,
				}
			}
			dst = adi.GetAccountByName(rtm, s[0])
			if dst == nil {
				return adi.Response{
					Text: "user not found",
				}
			}
			if src == dst {
				return adi.Response{
					Text: "can't give points to yourself",
				}
			}
			src.Sub(n)
			dst.Add(n)
			return adi.Response{
				Text: fmt.Sprintf("%s points %d. your points: %d",
					s[0], dst.Balance(), src.Balance()),
				Charge: true,
			}
		})

	adi.RegisterFunc("duel",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "challenge somebody to get their points",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 2 {
				return adi.Response{
					Text: "syntax: duel [user] [points|all]",
				}
			}
			if s[0] == "pot" {
				return adi.Response{
					Text: fmt.Sprintf("can't duel %s", s[0]),
				}
			}
			var src, dst adi.Account
			src = &u.Points
			dst = adi.GetAccountByName(rtm, s[0])
			if dst == nil {
				return adi.Response{
					Text: "user not found",
				}
			}
			if src == dst {
				return adi.Response{
					Text: "can't duel yourself",
				}
			}
			if dst.Balance() == 0 {
				return adi.Response{
					Text: fmt.Sprintf("%s has no points", s[0]),
				}
			}
			var n adi.Points
			if s[1] == "all" {
				if src.Balance() == 0 {
					return adi.Response{
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
					return adi.Response{
						Text: "syntax: duel [user] [points|all]",
					}
				}
				if t == 0 {
					return adi.Response{
						Text: "Must be more than 0 points",
					}
				}
				if adi.Points(t) > src.Balance() {
					return adi.Response{
						Text: fmt.Sprintf(
							"not enough points. your points: %d",
							src.Balance()),
					}
				}
				if adi.Points(t) > dst.Balance() {
					return adi.Response{
						Text: fmt.Sprintf(
							"%s does not have enough points. %s points: %d",
							s[0], s[0], dst.Balance()),
					}
				}
				n = adi.Points(t)
			}
			var t string
			if adi.RandBool() {
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
			return adi.Response{
				Text:   t,
				Charge: true,
			}
		})

	adi.RegisterFunc("pts",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text:   fmt.Sprintf("your points: %d", u.Points),
					Charge: true,
				}
			}
			src := adi.GetAccountByName(rtm, text)
			if src == nil {
				return adi.Response{
					Text: "user not found",
				}
			}
			return adi.Response{
				Text:   fmt.Sprintf("%s points: %d", text, src.Balance()),
				Charge: true,
			}
		})

	adi.RegisterFunc("trpts",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "transfer points",
				}
			}
			s := strings.Split(text, " ")
			if len(s) != 3 {
				return adi.Response{
					Text: "syntax: trpts [src] [dst] [points|all]",
				}
			}
			src := adi.GetAccountByName(rtm, s[0])
			if src == nil {
				return adi.Response{
					Text: fmt.Sprintf("user %s not found", s[0]),
				}
			}
			n, err := adi.ParsePoints(src, s[0], s[2])
			if err != "" {
				return adi.Response{
					Text: err,
				}
			}
			dst := adi.GetAccountByName(rtm, s[1])
			if dst == nil {
				return adi.Response{
					Text: fmt.Sprintf("user %s not found", s[1]),
				}
			}
			if src == dst {
				return adi.Response{
					Text: "source and destination can not be the same",
				}
			}
			src.Sub(n)
			dst.Add(n)
			return adi.Response{
				Text: fmt.Sprintf("%s points are now %d. %s points are now %d",
					s[0], src.Balance(), s[1], dst.Balance()),
				Charge: true,
			}
		})

	adi.RegisterFunc("cost",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			if text == "" {
				return adi.Response{
					Text: "find out the price of a command",
				}
			}
			cmd := adi.GetCommandByName(text)
			if cmd == nil {
				return adi.Response{
					Text: "command not found",
				}
			}
			return adi.Response{
				Text:   fmt.Sprintf("%s costs %d", text, cmd.Price),
				Charge: true,
			}
		})

	adi.RegisterFunc("lottery",
		func(text string, u *adi.User, rtm *slack.RTM) adi.Response {
			lot := &adi.GlobalBank.Lottery
			if text == "" {
				return adi.Response{
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
				return adi.Response{
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
					return adi.Response{
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
				return adi.Response{
					Text: t,
				}
			}
			var src adi.Account = &u.Points
			var n uint64
			{
				if text == "all" {
					if src.Balance() < lot.TicketPrice {
						return adi.Response{
							Text: "you do not have enough points.",
						}
					}
					n = uint64(src.Balance() / lot.TicketPrice)
				} else {
					t, err := strconv.ParseUint(text, 10, 64)
					if err != nil {
						return adi.Response{
							Text: "syntax: lottery [tickets|all]",
						}
					}
					if t == 0 {
						return adi.Response{
							Text: "needs to be at least 1",
						}
					}
					if adi.MulOverflows(uint64(lot.TicketPrice), t) ||
						lot.TicketsSold > (math.MaxUint64-t) {
						return adi.Response{
							Text: "can't buy that much tickets",
						}
					}
					n = t
				}
			}
			p := lot.TicketPrice * adi.Points(n)
			if p > src.Balance() {
				return adi.Response{
					Text: "you do not have enough points.",
				}
			}
			ts, ok := lot.Tickets[u.ID]
			if ok {
				if ts > (math.MaxUint64 - n) {
					return adi.Response{
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
			return adi.Response{
				Text: fmt.Sprintf("you bought %d tickets for %d. your points:%d. pot: %d",
					n, p, src.Balance(), lot.Pot.Balance(),
				),
				Charge: true,
			}
		})
}
