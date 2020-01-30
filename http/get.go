package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func main() {

	url := "http://127.0.0.1:8080/bar"

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Get failed:", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("statuscode:", resp.StatusCode)

	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Read failed:", err)
	}

	log.Println("content:", string(content))

}
