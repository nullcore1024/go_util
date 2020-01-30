package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {

	url := "http://127.0.0.1:8080/bar"
	contentType := "application/json;charset=utf-8"

	b := []byte("Hello, Server")
	body := bytes.NewBuffer(b)

	resp, err := http.Post(url, contentType, body)
	if err != nil {
		log.Println("Post failed:", err)
		return
	}

	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Read failed:", err)
		return
	}

	log.Println("content:", string(content))
}
