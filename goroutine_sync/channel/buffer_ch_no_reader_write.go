package main

func main() {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 1
	ch <- 1
	ch <- 1 //这一行操作就会发生阻塞，因为前三行的放入数据的操作已经把channel填满了
}
