package main

import "fmt"

func main() {
	ch := make(chan int, 5)
	ch <- 1
	ch <- 1
	close(ch)
	c := <-ch //只要channel还有数据，就可能执行取出操作
	fmt.Println(c)
	//正常结束
}
