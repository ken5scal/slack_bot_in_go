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
var WebHookURL = "https://hooks.slack.com/services/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"

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

	cmd := "/Users/suzuki/workspace/go/bin/lastpass_provisioning"
	out, err := exec.Command(cmd, "get", "users", "-f", "admin").Output()
	if err != nil {
		fmt.Println("failed command")
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	/*
	Incoming Web Hook
	 */
	//payload := &SlackMessage{Text:string(out)}
	//body := new(bytes.Buffer)
	//err = json.NewEncoder(body).Encode(payload)
	//if err != nil {
	//	fmt.Println("failed encoding")
	//	os.Exit(1)
	//}
	//url := "https://hooks.slack.com/services/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"
	//req, err = http.NewRequest(http.MethodPost, url, body)
	//if err != nil {
	//	fmt.Println("failed making request")
	//	os.Exit(1)
	//}
	//req.Header.Add("Content-Type", "application/json")
	//dump, _ := httputil.DumpRequest(req, true)
	//log.Println(dump)
	//
	//resp, err := http.DefaultClient.Do(req)
	//if err != nil {
	//	fmt.Println("failed requsting")
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//dump, _ = httputil.DumpResponse(resp, true)
	//fmt.Println(dump)
	//
	payload := &SlackSlashCommandResponse{
		ResponseType: "in_channel",
		Text: "Lastpass Audit",
		Attachments: make([]SlackAttachment, 0),
	}

	attachment := SlackAttachment{
		Color: "#36a64f",
		Pretext:"Admin users in LastPass",
		AuthorName:"LastPass Provisioning API",
		AuthorLink:"https://enterprise.lastpass.com/users/set-up-create-new-user-2/lastpass-provisioning-api/",
		AuthorIcon:"https://images-na.ssl-images-amazon.com/images/I/312B68fn10L.png",
		Title:"kengo-admin@moneyforward.co.jp",
		Text:"Activities",
		Fields:[]SlackField{
			{
				Title:"2017-08-25",
				Value:"従業員のアカウントを作成しました",
				Short:false,
			},
		},
	}
	payload.Attachments = []SlackAttachment{attachment}
	body := new(bytes.Buffer)
	err = json.NewEncoder(body).Encode(payload)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_, err = http.Post(vs["response_url"][0], "application/json", body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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

// Try Here: https://api.slack.com/docs/messages/builder

// SlackSimpleMessage is a simplest format of Slack message.
type SlackMessage struct {
	Text string `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	// _text_ for italy
	// *text* for bold
	// `text` for code block
}

type SlackAttachment struct {
	Fallback   string `json:"fallback"`
	Color      string `json:"color"`	//ex. "#36a64f" (green)
	Pretext    string `json:"pretext"`
	AuthorName string `json:"author_name"`
	AuthorLink string `json:"author_link"`	// "http://flickr.com/bobby/"
	AuthorIcon string `json:"author_icon"`	// "http://flickr.com/icons/bobby.jpg"
	Title      string `json:"title"`
	TitleLink  string `json:"title_link"`
	Text       string `json:"text"`
	Fields     []SlackField `json:"fields"`
	ImageURL   string `json:"image_url"`
	ThumbURL   string `json:"thumb_url"`
	Footer     string `json:"footer"`
	FooterIcon string `json:"footer_icon"`
	Ts         int    `json:"ts"`
	CallbackID     string `json:"callback_id"`
	AttachmentType string `json:"attachment_type"`
	Actions        []SlackAction `json:"actions,omitempty"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type SlackAction struct{
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
	ResponseType string `json:"response_type"`
	Text         string `json:"text"`
	Attachments  []SlackAttachment `json:"attachments"`
}