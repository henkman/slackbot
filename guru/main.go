package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

func main() {
	var config struct {
		Key string `json:"key"`
	}
	{
		fd, err := os.OpenFile("./config.json", os.O_RDONLY, 0600)
		if err != nil {
			panic(err)
		}
		if err := json.NewDecoder(fd).Decode(&config); err != nil {
			fd.Close()
		}
		fd.Close()
	}
	{
		fd, err := os.OpenFile("./log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0750)
		if err != nil {
			panic(err)
		}
		defer fd.Close()
		log.SetOutput(fd)
	}
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	api := slack.New(config.Key)
	api.SetDebug(false)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectedEvent:
			case *slack.MessageEvent:
				s := fmt.Sprintf("<@%s>", rtm.GetInfo().User.ID)
				if !strings.Contains(ev.Text, s) {
					continue Loop
				}
				texts := []string{
					"brot",
					"gar nicht",
					"hui, keine ahnung. da kenne ich mich gar nicht mit aus",
					"Yeah",
					"hui",
					"ja, sowas",
					"jaeoll",
					"jap",
					"kau.christoph",
					"kein problem",
					"moin",
					"morgen, ich bin heute in nem ganztages-meeting",
					"naja, fast gut",
					"weiß ich nicht genau",
					"wtf. break ist böse",
					"und ihr meint spring ist kompliziert und nicht nachvollziehbar. hier ist ein großer konkurrent",
					"die probleme hat man mit java nicht :)",
					"das ist doch alles kacke",
					"wat für ein ding?",
					"aso",
					"das passt nicht mit meiner struktur",
					"ALSO ICH BIN DABEI\nups",
					"hmm, ist schon verlockend",
					"wie groß ist er denn?",
					"ich mag keine schokolade :)",
					"Komme",
					"lol",
					"gerne, habe aber erst morgen dafür zeit",
					"Ok. Sag Bescheid wenn du mich noch brauchst.",
					"weiß ich nicht genau",
					"ja, was gibts?",
					"Ne, nur Fehler über Fehler",
					"jetzt, ja",
					"Naja, ich finde die Syntax noch sehr gewöhnungsbedürftig",
					"nein",
					"naja",
					"hmm, ok",
					"wat? ok",
					"Ich bin in Ulm. Haben ein Meeting mit dem Team dort",
					"ich habe mir eigentlich was für heute gekocht. aber ist nicht so gut geworden :)",
					"warum?",
					"Ich fahre mit dem Zug",
					"Ca 5 Min",
					"ja, auch kreditkarten soweit ich weiß",
					"33",
					"Ne, danke",
					"31... :)",
					"Kann ich dir nur empfehlen",
					"ich bin msl essen",
					"nope, geht auch dann nicht",
					"ich denke nicht",
					"hui, ok",
					"klingt kompliziert",
					"das passt ja schonmal",
					"ja, hast recht. macht absolut sinn",
					"da passiert einfach gar nix wenn man auf build klickt",
					"aha, cool",
					"nein, das spezialding wofür du den überhaupt geschrieben hast",
					"4",
					"schade",
					"was ist denn da der unterschied? es wird doch immer was dazu geschrieben. nur anstatt abbreviation locale",
					"ja, alles super. bei dir auch?",
					"hui, schä",
					"lool, ich hab gerade mein \"verlorenes\" token wieder gefunden. ich hab eben gedacht, meine tastatur ist aber wackelig. das token lag drunter... :)",
					"ja, die sind sehr hinterhältig",
					"lol, also ist das problem bekannt",
					"komisch",
					"wie kann eine font rotieren?",
					"ja, und was wenn der wert negativ ist?",
					"hmm",
					"hmm, ok",
					"wat? ne",
					"so, jetzt müsste es passen",
					"gockel, lustiger name :)",
					"das klappt nicht, habe ich letztens auch versucvht :)",
					"ja, irgendwas stimmt nicht",
					"genau",
					"lolz",
					"mjam, lecker",
					"ja, macht sinn",
					"Hmm, das kann natürlich durchaus sein",
					"Du kannst tanzen? Das ist mir neu :)",
					"Benutzt Spring, das ist geil!",
					"Hi. Ich habe mir den Fuß ungeknickt und bin krank geschrieben",
					"Genau, fast",
				}
				c := texts[rand.Int31n(int32(len(texts)))]
				rtm.SendMessage(
					rtm.NewOutgoingMessage(c, ev.Channel))
			case *slack.PresenceChangeEvent:
			case *slack.LatencyReport:
			case *slack.RTMError:
				log.Printf("Error: %s\n", ev.Error())
			case *slack.InvalidAuthEvent:
				log.Println("Invalid credentials")
				break Loop
			default:
			}
		}
	}
}
