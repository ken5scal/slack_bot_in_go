package main

import (
	"net/http"
	"fmt"
	"net/url"

	"log"
	"sync"
	"os"
	"encoding/json"
	"io/ioutil"

	"bytes"
	"os/exec"
)

var PORT = "4390"
var clientId, clientSecret, token string

func init() {
	clientId = os.Getenv("clientId")
	clientSecret = os.Getenv("clientSecret")
	token = os.Getenv("token")
}

func main() {
	if len(clientId) == 0 || len(clientSecret) == 0  {
		log.Fatalln("Either clientId or clientSecret is empty")
	}

	//http.ListenAndServeTLS()
	http.HandleFunc("/", rootDir)
	http.HandleFunc("/oauth", oauth)
	http.HandleFunc("/command", command)
	http.HandleFunc("/listening", listening)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func () {
		defer wg.Done()
		log.Printf("Listening on port: %v\n", PORT)
		log.Fatal(http.ListenAndServe(":"+PORT, nil))
	}()
	wg.Wait()
}

func rootDir(res http.ResponseWriter, req *http.Request) {
	data := struct {
		Method string
		URL *url.URL
	}{
		req.Method,
		req.URL,
	}

	fmt.Fprintln(res, "Ngrok is working! -  Path Hit: " + data.URL.Host + data.URL.Path)
}

// GET /oauth?code=somekindofcode
func oauth(res http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		http.Error(res, "Failed parsing form", http.StatusInternalServerError)
	}

	code := req.Form.Get("code")
	if code == "" {
		errorModel := struct {
			Error string
		}{
			"Looks like we're not getting code.",
		}

		js, err := json.Marshal(errorModel)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(js)
	} else {
		// We'll do a GET call to Slack's `oauth.access` endpoint, passing our app's client ID, client secret, and the code we just got as query parameters.
		oauthRequest := "https://slack.com/api/oauth.access?" +
			"code=" + code + "&" +
			"client_id=" + clientId + "&" +
			"client_secret=" + clientSecret
		oauthResponse, err := http.Get(oauthRequest)
		if err != nil {
			http.Error(res, "Failed fetching oauth request", http.StatusInternalServerError)
		} else {
			body, _ := ioutil.ReadAll(oauthResponse.Body)
			defer oauthResponse.Body.Close()
			res.Write(body)
			command(res, req)
		}
	}
}

func command(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(res, `{"response_type": "ephemeral","text": "","attachments": []}`)

	buf := new(bytes.Buffer)
	defer req.Body.Close()

	buf.ReadFrom(req.Body)
	fmt.Println(buf.String())
	vs, err := url.ParseQuery(buf.String())
	if err != nil {
		os.Exit(1)
	}

	out, err := exec.Command("/Users/suzuki/workspace/go/bin/lastpass_provisioning", "dashboard").Output()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for key, value := range vs {
		if key == "response_url" {
			type Text struct {
				Text         string `json:"text"`
			}

			type SlackJson struct {
				ResponseType string `json:"response_type"`
				Text         string `json:"text"`
				Attachments  []Text `json:"attachments"`
			}

			payload := &SlackJson{
				ResponseType: "in_channel",
				Text: "Lastpass Audit",
				Attachments:[]Text{{string(out)}},
			}

			body := new(bytes.Buffer)
			err := json.NewEncoder(body).Encode(payload)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			_, err = http.Post(value[0], "application/json", body)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
}

func listening(res http.ResponseWriter, req *http.Request) {
	// ============= Slack URL Verification ============
	// In order to verify the url of our endpoint, Slack will send a challenge
	// token in a request and check for this token in the response our endpoint sends back.
	// For more info: https://api.slack.com/events/url_verification
	var ch ChallengeBody
	err := json.NewDecoder(req.Body).Decode(&ch)
	if err != nil {
		log.Fatalln("error unmarshalling", err)
	}

	fmt.Println(ch)

	if ch.Challenge != "" {
		b, err := json.Marshal(ChallengeResponse{ch.Challenge})
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		res.Write(b)
		return
	}

	// ============ Slack Token Verification ===========
	// We can verify the request is coming from Slack by checking that the
	// verification token in the request matches our app's settings
	if token != ch.Token {
		res.Header().Set("X-Slack-No-Retry", "1")
		res.WriteHeader(http.StatusForbidden)
	}

	// ====== Process Incoming Events from Slack =======
	// If the incoming request is an Event we've subcribed to
}

type ChallengeResponse struct {
	Challenge string
}

type ChallengeBody struct {
	Token string
	Team_id string
	Api_app_id string
	Challenge string
	Type string
	Event_ts string
	Event Event
}

type Event struct {
	Type string
	Event_ts string
	User string
	Reaction string
	Item_user string
	Item struct{
		Type string
		Channel string
		Ts string
	}
}

type SlackSimpleMessage struct {
	Text string `json:"text"`
}

// SlackMessageWithAttachments is a format for Slack message.
// https://api.slack.com/docs/messages/builder?msg=%7B%22attachments%22%3A%5B%7B%22fallback%22%3A%22Required%20plain-text%20summary%20of%20the%20attachment.%22%2C%22color%22%3A%22%2336a64f%22%2C%22pretext%22%3A%22Optional%20text%20that%20appears%20above%20the%20attachment%20block%22%2C%22author_name%22%3A%22Bobby%20Tables%22%2C%22author_link%22%3A%22http%3A%2F%2Fflickr.com%2Fbobby%2F%22%2C%22author_icon%22%3A%22http%3A%2F%2Fflickr.com%2Ficons%2Fbobby.jpg%22%2C%22title%22%3A%22Slack%20API%20Documentation%22%2C%22title_link%22%3A%22https%3A%2F%2Fapi.slack.com%2F%22%2C%22text%22%3A%22Optional%20text%20that%20appears%20within%20the%20attachment%22%2C%22fields%22%3A%5B%7B%22title%22%3A%22Priority%22%2C%22value%22%3A%22High%22%2C%22short%22%3Afalse%7D%5D%2C%22image_url%22%3A%22http%3A%2F%2Fmy-website.com%2Fpath%2Fto%2Fimage.jpg%22%2C%22thumb_url%22%3A%22http%3A%2F%2Fexample.com%2Fpath%2Fto%2Fthumb.png%22%2C%22footer%22%3A%22Slack%20API%22%2C%22footer_icon%22%3A%22https%3A%2F%2Fplatform.slack-edge.com%2Fimg%2Fdefault_application_icon.png%22%2C%22ts%22%3A123456789%7D%5D%7D
/*
{
    "attachments": [
        {
            "fallback": "Required plain-text summary of the attachment.",
            "color": "#36a64f",
            "pretext": "Optional text that appears above the attachment block",
            "author_name": "Bobby Tables",
            "author_link": "http://flickr.com/bobby/",
            "author_icon": "http://flickr.com/icons/bobby.jpg",
            "title": "Slack API Documentation",
            "title_link": "https://api.slack.com/",
            "text": "Optional text that appears within the attachment",
            "fields": [
                {
                    "title": "Priority",
                    "value": "Low",
                    "short": false
                }
            ],
            "image_url": "https://platform.slack-edge.com/img/default_application_icon.png",
            "thumb_url": "https://platform.slack-edge.com/img/default_application_icon.png",
            "footer": "Slack API",
            "footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
            "ts": 123456789
        }
    ]
}
 */
type SlackMessageWithAttachments struct {
	Attachments []struct {
		Fallback   string `json:"fallback"`
		Color      string `json:"color"`	//ex. "#36a64f" (green)
		Pretext    string `json:"pretext"`
		AuthorName string `json:"author_name"`
		AuthorLink string `json:"author_link"`	// "http://flickr.com/bobby/"
		AuthorIcon string `json:"author_icon"`	// "http://flickr.com/icons/bobby.jpg"
		Title      string `json:"title"`
		TitleLink  string `json:"title_link"`
		Text       string `json:"text"`
		Fields     []struct {
			Title string `json:"title"`
			Value string `json:"value"`
			Short bool   `json:"short"`
		} `json:"fields"`
		ImageURL   string `json:"image_url"`
		ThumbURL   string `json:"thumb_url"`
		Footer     string `json:"footer"`
		FooterIcon string `json:"footer_icon"`
		Ts         int    `json:"ts"`
	} `json:"attachments"`
}