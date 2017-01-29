package main

import (
	"net/http"
	"fmt"
	"net/url"
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
	http.ListenAndServe(PORT, hoge)
}
