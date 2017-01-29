package main

import (
	"net/http"
	"fmt"
	"net/url"

	"log"
	"sync"
	"os"
	"encoding/json"
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

func oauth(res http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		log.Fatalln("Failed Parsing Form")
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
	}
}

func command(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Your ngrok tunnel is up and running!")
}

type Response struct {
	Message string `json:"message"`

}