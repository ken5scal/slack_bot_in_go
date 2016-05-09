package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

const slackUrl = "https://hooks.slack.com/services/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"

type Slack struct {
	Text string `json:"text"`
}

func main() {
	params, _ := json.Marshal(Slack{Text: "hogehoge"})
	values := url.Values{}
	values.Add("payload", string(params))

	println(string(params))

	resp, _ := http.PostForm(slackUrl, values)
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	println(string(body))
}
