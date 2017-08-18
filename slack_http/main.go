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
	fmt.Fprintln(res, `{"response_type": "in_channel","text": "It's 80 degrees right now.","attachments": [{"text":"Partly cloudy today and tomorrow"}]}`)

	//out, err := exec.Command("/Users/suzuki/workspace/go/bin/lastpass_provisioning", "dashboard").Output()
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//fmt.Println(string(out))
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