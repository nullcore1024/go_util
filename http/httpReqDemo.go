package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	//"testing"
)

func getHttpReq(r *bufio.Reader) (*http.Request, error) {
	return http.ReadRequest(r)
}

func testGet(data string) (consumer int) {
	r := bufio.NewReader(strings.NewReader((data)))
	if req, err := getHttpReq(r); err == nil {
		fmt.Println("method", req.Method)
		fmt.Println("url", req.URL)
		fmt.Println("content-length", req.ContentLength)
		if "POST" == req.Method {
			body, err := ioutil.ReadAll(req.Body)
			fmt.Println("err:", err)
			fmt.Println("body:", string(body))
			if err == nil && req.ContentLength != int64(len(body)) {
				return -1
			}
		}
		return r.Buffered()
	}

	return -1
}

func main() {
	get := "GET /abb/ff HTTP/1.1\r\n\r\n"
	get2 := "GET /abc/dd HTTP/1.1\r\n\r\nGET"
	fmt.Println("get len", len(get))
	fmt.Println("get len consumer", testGet(get))
	fmt.Println("get2 len", len(get2))
	fmt.Println("get2 len consumer", testGet(get2))

	fmt.Println("---------------------")
	post := "POST /abc/ee HTTP/1.1\r\nContent-Length: 3\r\n\r\nGET"
	fmt.Println("post len", len(post))
	fmt.Println("post len consumer", testGet(post))

	post = "POST /abc/ee HTTP/1.1\r\nContent-Length: 3\r\n\r\n"
	fmt.Println("post2 len", len(post))
	fmt.Println("post2 len consumer", testGet(post))
}
