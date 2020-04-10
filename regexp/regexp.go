package main

import (
	"fmt"
	"regexp"
)

func Match(reg, dst string) {
	allowed, err := regexp.Match(reg, []byte(dst))
	if !allowed || err != nil {
		fmt.Println("err", err)
	} else {
		fmt.Println("match", dst)
	}
}

func main() {
	allow := "coins_log.log"
	path := allow
	Match(allow, path)
	Match("c*.log", "ccc.log")
}
