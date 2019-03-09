package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/nlopes/slack"
)

type ImageType string

const (
	ImageType_All     = ""
	ImageType_Clipart = "clipart"
	ImageType_Face    = "face"
	ImageType_Lineart = "lineart"
	ImageType_News    = "news"
	ImageType_Photo   = "photo"
)

func img(text string, m *slack.MessageEvent, rtm *slack.RTM) {
	response, err := googleImageSearch(text, ImageType_All, true, RandUint32(100)+1, 1)
	if err != nil {
		log.Println(err)
		rtm.SendMessage(rtm.NewOutgoingMessage("internal error", m.Channel))
		return
	}
	if len(response) > 0 {
		rtm.SendMessage(rtm.NewOutgoingMessage(response[0], m.Channel))
	} else {
		rtm.SendMessage(rtm.NewOutgoingMessage("nothing found", m.Channel))
	}
}

func googleImageSearch(query string, t ImageType, safe bool, start, count uint32) ([]string, error) {
	ps := url.Values{
		"q":          []string{query},
		"key":        []string{config.Google.Key},
		"cx":         []string{config.Google.CSE},
		"num":        []string{fmt.Sprint(count)},
		"start":      []string{fmt.Sprint(start)},
		"searchType": []string{"image"},
	}
	if t != ImageType_All {
		ps.Set("imgType", string(t))
	}
	if safe {
		ps.Set("safe", "active")
	} else {
		ps.Set("safe", "off")
	}
	url_str := "https://www.googleapis.com/customsearch/v1?" + ps.Encode()
	resp, err := client.Get(url_str)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []struct {
			Link string `json:"link"`
		} `json:"items"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	links := make([]string, len(result.Items))
	for i, _ := range result.Items {
		links[i] = result.Items[i].Link
	}
	return links, nil
}
