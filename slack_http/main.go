package main

import (
	"net/http"
	"fmt"
	"net/url"

	"log"
	"sync"
)

var PORT = "4390"

type Hoge int


func (h Hoge) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data := struct {
		Method string
		URL *url.URL
	}{
		req.Method,
		req.URL,
	}

	fmt.Fprintln(w, "Ngrok is working! -  Path Hit: " + data.URL.Host + data.URL.Path)
}

func main() {
	//http.ListenAndServeTLS()
	var hoge Hoge

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func () {
		defer wg.Done()
		log.Printf("Listening on port: %v\n", PORT)
		log.Fatal(http.ListenAndServe(":"+PORT, hoge))
	}()
	wg.Wait()
}
