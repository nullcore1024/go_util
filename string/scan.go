package main

import (
	"fmt"
)

func Parse(input string) (all []string, err error) {
	prev := 0
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case ',', '/', '#':
			all = append(all, input[prev:i+1])
			prev = i + 1
		}
	}
	return
}

func main() {
	const input = "232323231.r/2223333#2abc-d,3,4,5#6.7?"

	all, _ := Parse(input)
	for _, v := range all {
		fmt.Printf("%s\n", v)
	}
}
