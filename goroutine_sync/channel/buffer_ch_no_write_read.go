package main

func main() {
	ch := make(chan int, 3)
	<-ch
}
