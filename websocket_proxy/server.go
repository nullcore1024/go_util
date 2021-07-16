package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fvbock/endless"
	"github.com/gorilla/mux"
	websocket "github.com/gorilla/websocket"
	log "google.golang.org/grpc/grpclog"
)

const (
	kReadTimeout = time.Second * 3
	kUrl         = "/ws"
)

var (
	addr = flag.String("addr", "localhost:48080", "websocket service address")
	peer = flag.String("peer", "localhost:48081", "peer address")

	upgrader = websocket.Upgrader{} // use default options
)

func handler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("upgrade fail, err:", err)
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
			case <-ctx.Done():
				log.Info("tcp peer close, time=", time.Now().Unix())
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
				log.Infof("dst read err:%s\n", err)
				break
			}
			err = src.WriteMessage(websocket.TextMessage, buf)
			if err != nil {
				log.Error("ws write src err:", err)
				break
			}
		}
		cancel()
	}(ctx, c, tcpConn)

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Error("ws read fail, err:", err)
			wg.Done()
			break
		}
		io.Copy(tcpConn, bytes.NewBuffer(message))
		select {
		case <-ctx.Done():
			log.Info("ws peer close, time=", time.Now().Unix())
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
	log.SetLoggerV2(log.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout))
	mux := mux.NewRouter()
	mux.HandleFunc(kUrl, handler)
	err := endless.ListenAndServe(*addr, mux)
	if err != nil {
		log.Error(err)
	}
}
