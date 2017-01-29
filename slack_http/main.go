package main

import (
	"net/http"
	"fmt"
	"net/url"

	"log"
	"sync"
	"os"
)

var PORT = "4390"

func hoge(res http.ResponseWriter, req *http.Request) {
	data := struct {
		Method string
		URL *url.URL
	}{
		req.Method,
		req.URL,
	}

	fmt.Fprintln(res, "Ngrok is working! -  Path Hit: " + data.URL.Host + data.URL.Path)
}


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
	http.HandleFunc("/", hoge)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func () {
		defer wg.Done()
		log.Printf("Listening on port: %v\n", PORT)
		log.Fatal(http.ListenAndServe(":"+PORT, nil))
	}()
	wg.Wait()
}
