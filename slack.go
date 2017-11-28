package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"net/url"
)

/**
Slash Command Section
https://api.slack.com/slash-commands
*/
type SlashCommandRequest struct {
	Token       string
	TeamID      string
	TeamDomain  string
	ChannelID   string
	ChannelName string
	UserID      string
	UserName    string
	Command     string
	Text        string
	ResponseURL string
	TriggerID   string
}

func BuildSlashCommandRequestFromQuery(query string) (*SlashCommandRequest, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	vs, err := url.ParseQuery(query)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to Parse Query from slack: %s", err))
	}

	if _, ok := vs["token"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses token", query))
	} else if _, ok := vs["team_id"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses team_id", query))
	} else if _, ok := vs["team_domain"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses team_domain", query))
	} else if _, ok := vs["channel_id"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses channel_id", query))
	} else if _, ok := vs["user_id"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses user_id", query))
	} else if _, ok := vs["user_name"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses user_name", query))
	} else if _, ok := vs["command"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses command", query))
	} else if _, ok := vs["response_url"]; !ok {
		return nil, errors.New(fmt.Sprintf("query `%s` misses response_url", query))
	}

	return &SlashCommandRequest{
		Token:       vs["token"][0],
		TeamID:      vs["team_id"][0],
		TeamDomain:  vs["team_domain"][0],
		ChannelID:   vs["channel_id"][0],
		UserID:      vs["user_id"][0],
		UserName:    vs["user_name"][0],
		Command:     vs["command"][0],
		Text:        vs["text"][0],
		ResponseURL: vs["response_url"][0],
	}, nil
}

func (s *SlashCommandRequest) sendResponse(responseText string) error {
	body := new(bytes.Buffer)
	payload := &SlackSlashCommandResponse{
		ResponseType: "in_channel", //ephemeral
		Text:         responseText,
		Attachments:  make([]SlackAttachment, 0),
	}

	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return errors.New(fmt.Sprintf("Failed to encode paylod %v: %s", payload, err))
	}

	if _, err := http.Post(s.ResponseURL, "application/json", body); err != nil {
		return err
	}

	return nil
}

// Try Here: https://api.slack.com/docs/messages/builder

// SlackSimpleMessage is a simplest format of Slack message.
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	// _text_ for italy
	// *text* for bold
	// `text` for code block
}

type SlackAttachment struct {
	Fallback       string        `json:"fallback"`
	Color          string        `json:"color"` //ex. "#36a64f" (green)
	Pretext        string        `json:"pretext"`
	AuthorName     string        `json:"author_name"`
	AuthorLink     string        `json:"author_link"` // "http://flickr.com/bobby/"
	AuthorIcon     string        `json:"author_icon"` // "http://flickr.com/icons/bobby.jpg"
	Title          string        `json:"title"`
	TitleLink      string        `json:"title_link"`
	Text           string        `json:"text"`
	Fields         []SlackField  `json:"fields"`
	ImageURL       string        `json:"image_url"`
	ThumbURL       string        `json:"thumb_url"`
	Footer         string        `json:"footer"`
	FooterIcon     string        `json:"footer_icon"`
	Ts             int           `json:"ts"`
	CallbackID     string        `json:"callback_id"`
	AttachmentType string        `json:"attachment_type"`
	Actions        []SlackAction `json:"actions,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type SlackAction struct {
	Name    string `json:"name"`
	Text    string `json:"text"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Style   string `json:"style,omitempty"`
	Confirm struct {
		Title       string `json:"title"`
		Text        string `json:"text"`
		OkText      string `json:"ok_text"`
		DismissText string `json:"dismiss_text"`
	} `json:"confirm,omitempty"`
}

type SlackSlashCommandResponse struct {
	ResponseType string            `json:"response_type"`
	Text         string            `json:"text"`
	Attachments  []SlackAttachment `json:"attachments"`
}

type Event struct {
	Type      string
	Event_ts  string
	User      string
	Reaction  string
	Item_user string
	Item      struct {
		Type    string
		Channel string
		Ts      string
	}
}
