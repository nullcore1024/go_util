package main

import (
	"flag"
	"fmt"
	_ "io"
	"net"
	"os"
	"time"
)

var host = flag.String("host", "", "host")
var port = flag.String("port", "3333", "port")
var timeout = flag.Int("timeout", 0, "read timeout(default 0)")

func main() {
	flag.Parse()
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", *host+":"+*port)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on " + *host + ":" + *port)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}
		if *timeout > 0 {
			conn.SetReadDeadline(time.Now().Add(time.Duration(*timeout) * time.Second))
			fmt.Printf("now:%v, read deadline:%v\n", time.Now(), time.Now().Add(time.Duration(*timeout)*time.Second))
		}
		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}
}
func handleRequest(conn net.Conn) {
	defer conn.Close()
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("now:%v, read err:%s\n", time.Now(), err)
			return
		}
		conn.Write(buf)
		//io.Copy(conn, conn)
	}
}
