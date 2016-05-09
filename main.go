package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

const slackUrl = "https://hooks.slack.com/services/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"

type Slack struct {
	Text      string `json:"text"`
	UserName  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
}

var slack Slack

func main() {
	slack := Slack{
		Text:      "Hello, this is gopher",
		IconEmoji: ":gopher:",
		UserName:  "InstantsBot",
	}
	params, _ := json.Marshal(slack)
	values := url.Values{}
	values.Add("payload", string(params))

	println(string(params))

	resp, _ := http.PostForm(slackUrl, values)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	println(string(body))
}
