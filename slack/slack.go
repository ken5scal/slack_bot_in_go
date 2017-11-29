package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	defaultURL = "https://slack.com/api/"
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

type SlashCommandResponse struct {
	ResponseType string       `json:"response_type"`
	Text         string       `json:"text"`
	Attachments  []Attachment `json:"attachments"`
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

func (s *SlashCommandRequest) SendResponse(responseText string) error {
	body := new(bytes.Buffer)
	payload := &SlashCommandResponse{
		ResponseType: "in_channel", //ephemeral
		Text:         responseText,
		Attachments:  make([]Attachment, 0),
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
type Message struct {
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	// _text_ for italy
	// *text* for bold
	// `text` for code block
}

type Attachment struct {
	Fallback       string   `json:"fallback"`
	Color          string   `json:"color"` //ex. "#36a64f" (green)
	Pretext        string   `json:"pretext"`
	AuthorName     string   `json:"author_name"`
	AuthorLink     string   `json:"author_link"` // "http://flickr.com/bobby/"
	AuthorIcon     string   `json:"author_icon"` // "http://flickr.com/icons/bobby.jpg"
	Title          string   `json:"title"`
	TitleLink      string   `json:"title_link"`
	Text           string   `json:"text"`
	Fields         []Field  `json:"fields"`
	ImageURL       string   `json:"image_url"`
	ThumbURL       string   `json:"thumb_url"`
	Footer         string   `json:"footer"`
	FooterIcon     string   `json:"footer_icon"`
	Ts             int      `json:"ts"`
	CallbackID     string   `json:"callback_id"`
	AttachmentType string   `json:"attachment_type"`
	Actions        []Action `json:"actions,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type Action struct {
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

/**
Client is a client for slack
*/
type Client struct {
	URL       *url.URL
	Token     string
	Verbose   bool
	UserAgent string
	Logger    *log.Logger
}

func NewClient(accessToken string, verbose bool) (*Client, error) {
	parsedURL, err := url.ParseRequestURI(defaultURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		URL:     parsedURL,
		Token:   accessToken,
		Verbose: verbose,
		Logger:  nil,
	}, nil
}

func debugRequest(req *http.Request) {
	dump, err := httputil.DumpRequest(req, true)
	if err == nil {
		log.Printf("%s", dump)
	}
}

func debugResponse(resp *http.Response) {
	dump, err := httputil.DumpResponse(resp, true)
	if err == nil {
		log.Printf("%s", dump)
	}
}

// ChatPostMessage posts a message in designated channel
func (s *Client) ChatPostMessage(message, channel string) (*ChatPostResponse, error) {
	data := &ChatPostMessage{Channel: channel, Text: message}
	payload, err := json.Marshal(data)

	if err != nil {
		return nil, fmt.Errorf("Failed to encode %v into JSON\n", data)
	}

	// Form a request.
	// req, err := http.Post(s.URL.String(), "application/json", bytes.NewBuffer(body))
	req, err := http.NewRequest(http.MethodPost, s.URL.String()+"chat.postMessage", bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+s.Token)

	if s.Verbose {
		debugRequest(req)
	}

	// Start request
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	if s.Verbose {
		debugResponse(resp)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to read response body: %s", resp.Body))
	}

	var response ChatPostResponse
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to decode json message from slack: %s", body))
	}

	if !response.Ok {
		return nil, errors.New(response.Error)
	}

	return &response, nil
}

type ChatPostMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
	//Pretty int `json:"pretty"`
	Attachments []struct {
		Text           string `json:"text"`
		Fallback       string `json:"fallback"`
		Color          string `json:"color"`
		AttachmentType string `json:"attachment_type"`
		CallbackID     string `json:"callback_id"`
		Actions        []struct {
			Name       string `json:"name"`
			Text       string `json:"text"`
			Type       string `json:"type"`
			DataSource string `json:"data_source"`
		} `json:"actions"`
	} `json:"attachments"`
}

type ChatPostResponse struct {
	Ok        bool   `json:"ok"`
	Channel   string `json:"channel"`
	Error     string `json:"error"`
	Warning   string `json:"warning"`
	TimeStamp string `json:"ts"`
	Message   struct {
		Text     string `json:"text"`
		Username string `json:"username"`
		BotID    string `json:"bot_id"`
		Type     string `json:"type"`
		Subtype  string `json:"subtype"`
		Ts       string `json:"ts"`
	} `json:"message"`
}
