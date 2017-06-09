package yahoo

import (
	"fmt"
	"log"
	"strings"

	"github.com/henkman/ipinfo"
	"github.com/henkman/slackbot/adi"
	"github.com/henkman/yahoo"
	"github.com/nlopes/slack"
)

var (
	session yahoo.Session
)

func formatWeather(wf yahoo.WeatherForecast) string {
	return fmt.Sprintf(":weather%s: _%s_ - %s - *%s/%s °C*",
		wf.Code,
		wf.Text,
		wf.Day,
		wf.High,
		wf.Low)
}

func init() {

	adi.RegisterFunc("weather",
		func(m adi.Message, rtm *slack.RTM) adi.Response {
			if !session.IsInitialized() {
				if err := session.Init(); err != nil {
					log.Println("yahoo api:", err.Error())
					return adi.Response{
						Text: "could not initialize weather api",
					}
				}
			}
			var location string
			text := strings.TrimSpace(m.Text)
			if text == "" {
				info, err := ipinfo.Query("")
				if err != nil {
					log.Println("ipinfo api:", err.Error())
					return adi.Response{
						Text: "could not geolocate",
					}
				}
				location = info.City + ", " + info.Country
			} else {
				location = text
			}
			wfs, err := session.GetWeatherForecast(
				location, 7, yahoo.TemperatureUnit_Celcius)
			if err != nil {
				log.Println("weather forecast failed", err.Error())
				return adi.Response{
					Text: "no results",
				}
			}
			n := len(wfs)
			if text == "" {
				n++
			}
			out := make([]string, 0, n)
			if text == "" {
				out = append(out, location)
			}
			for _, wf := range wfs {
				out = append(out, formatWeather(wf))
			}
			return adi.Response{
				Text:   strings.Join(out, "\n"),
				Charge: true,
			}
		})
}
