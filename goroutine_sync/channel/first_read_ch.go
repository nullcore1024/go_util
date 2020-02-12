package main

import (
	"fmt"
	"sync"
)

/*
  无缓冲通道与有缓冲通道的主要区别为：无缓冲通道存取数据是同步的，即如果通道中无数据，则通道一直处于阻塞状态；有缓冲通道存取数据是异步的，即存取数据互不干扰，只有当通道中已满时，存数据操作，通道阻塞；当通道中为空时，取数据操作，通道阻塞。

   因此，使用无缓冲的channel时，放入操作和取出操作不能在同一个routine中，而且应该是先确保有某个routine对它执行取出操作，然后才能在另一个routine中执行放入操作，否则会发生死锁现象
*/

var waitGroup sync.WaitGroup //使用wg等待所有routine执行完毕，并输出相应的提示信息

func AFunc(ch chan int) {
	waitGroup.Add(1)
FLAG:
	for {
		select {
		case val := <-ch:
			fmt.Println(val)
			break FLAG
		}
	}
	waitGroup.Done()
	fmt.Println("WaitGroup Done")
}

func main() {
	ch := make(chan int) //无缓冲通道
	execMode := 0        //执行模式 0：先启动并发，正常输出100 1：后启动并发，发生死锁
	switch execMode {
	case 0:
		go AFunc(ch)
		ch <- 100
	case 1:
		ch <- 100
		go AFunc(ch)
	}
	waitGroup.Wait()
	close(ch)
}
