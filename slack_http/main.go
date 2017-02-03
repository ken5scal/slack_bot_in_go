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
var clientId, clientSecret string

func init() {
	clientId = os.Getenv("clientId")
	clientSecret = os.Getenv("clientSecret")
}

func main() {
	if len(clientId) == 0 || len(clientSecret) == 0 {
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
	fmt.Fprintln(res, "Your ngrok tunnel is up and running!")
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

	b, err := json.Marshal(ChallengeResponse{ch.Challenge})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/x-www-form-urlencoded")
	res.Write(b)
}

type ChallengeResponse struct {
	Challenge string
}

type ChallengeBody struct {
	Token string
	Challenge string
	Type string
}