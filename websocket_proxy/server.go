package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	websocket "github.com/gorilla/websocket"
)

const kReadTimeout = time.Second * 3

var (
	addr = flag.String("addr", "localhost:48080", "http service address")
	peer = flag.String("peer", "localhost:48081", "peer address")
)

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)

	tcpConn, err := net.Dial("tcp", *peer)
	defer tcpConn.Close()

	go func(ctx context.Context, src *websocket.Conn, dst net.Conn) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done(): //取出值即说明是结束信号
				log.Println("tcp peer close, time=", time.Now().Unix())
				return
			default:
			}

			buf := make([]byte, 1024)
			if kReadTimeout.Seconds() != 0 {
				dst.SetReadDeadline(time.Now().Add(kReadTimeout))
			}
			_, err = dst.Read(buf)
			if err != nil {
				netErr, ok := err.(net.Error)
				if ok && netErr.Timeout() && netErr.Temporary() {
					continue // no data, not error
				}
				log.Printf("tcp read err:%s\n", err)
				break
			}
			log.Printf("dst recv msg:%s\n", buf)
			err = src.WriteMessage(websocket.TextMessage, buf)
			if err != nil {
				log.Println("ws write src err:", err)
				break
			}
		}
		cancel()
	}(ctx, c, tcpConn)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("ws read:", err)
			wg.Done()
			break
		}
		log.Printf("recv: %s\n", message)
		io.Copy(tcpConn, bytes.NewBuffer(message))
		select {
		case <-ctx.Done(): //取出值即说明是结束信号
			log.Print("ws peer close, time=", time.Now().Unix())
			wg.Done()
			return
		default:
		}
	}
	cancel()
	wg.Wait()
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
