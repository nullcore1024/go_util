package main

/*
关闭后的channel可以取数据，但是不能放数据。而且，channel在执行了close()后并没有真的关闭，channel中的数据全部取走之后才会真正关闭。
*/
func main() {
	ch := make(chan int, 5)
	ch <- 1
	ch <- 1
	close(ch)
	ch <- 1 //不能对关闭的channel执行放入操作
	// 会触发panic
}
