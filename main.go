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
		Text:      "*Hello*,\n _this is gopher_\n`radioactive` is `true`.",
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
