package main

import (
	"fmt"
)

func main() {
	fmt.Println("1")

Exit:
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			fmt.Println("j", j)
			if i+j > 3 {
				fmt.Print("exit")
				break Exit
			}
		}
		fmt.Println("i", i)
	}

	fmt.Println("3")
}
