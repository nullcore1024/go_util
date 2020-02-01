package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func recv(ctx context.Context, wg *sync.WaitGroup, ch chan []byte) bool {
	//这个是为了避免stopChan已经close但是下面的第二个select多次随机执行t.msgChan <- msg
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("done")
			wg.Done()
			return true
		case <-t.C:
			fmt.Println("ticker")
			continue
		case msg := <-ch:
			fmt.Println("sz ", len(msg))
			fmt.Println("msg", msg)
		}
	}
	wg.Done()
	return true
}

func main() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	ch := make(chan []byte)
	go recv(ctx, &wg, ch)
	ch <- []byte("hello")
	time.Sleep(3 * time.Second)
	cancel()
	fmt.Println("cancel")
	fmt.Println("close")
	close(ch)
	fmt.Println("close over")
	wg.Wait()
	fmt.Println("exit")
}
