package main

import (
	"fmt"
	"sync"
)

/*
使用带缓冲的channel时，因为有缓冲空间，所以只要缓冲区不满，放入操作就不会阻塞，同样，只要缓冲区不空，取出操作就不会阻塞。而且，带有缓冲的channel的放入和取出操作可以用在同一个routine中。但是，一定要注意放入和取出的速率问题，否则也会发生死锁现
*/

var waitGroup sync.WaitGroup

func AFunc(ch chan int, putMode int) {
	val := <-ch
	switch putMode {
	case 0:
		fmt.Printf("Vaule=%d\n", val)
	case 1:
		fmt.Printf("Vaule=%d\n", val)
		for i := 1; i <= 5; i++ {
			ch <- i * val
		}
	case 2:
		fmt.Printf("Vaule=%d\n", val)
		for i := 1; i <= 5; i++ {
			<-ch
		}
	}

	waitGroup.Done()
	fmt.Println("WaitGroup Done", val)
}

func main() {
	ch := make(chan int, 10)
	putMode := 0 //该模式下，能够正常输出所有数据
	//putMode := 1 //当放入速度远大于取数速度时，程序阻塞
	//putMode := 2 //当取数速度远大于放数速度时，程序阻塞
	for i := 0; i < 1000; i++ {
		ch <- i
		waitGroup.Add(1)
		go AFunc(ch, putMode)
	}
	waitGroup.Wait()
	close(ch)
}
