package main

import (
	"fmt"
	"net"
	"os"
)

func main() {

	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("dial failed:", err)
		os.Exit(1)
	}
	defer conn.Close()

	buffer := make([]byte, 512)

	for {

		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Read failed:", err)
			return
		}

		fmt.Println("count:", n, "msg:", string(buffer))
	}

}
