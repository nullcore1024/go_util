package main

import (
	"encoding/json"
	"fmt"
)

func map2string() string {
	var s map[string]string
	s = make(map[string]string)
	s["aa"] = "bbbyyp"
	s["cc"] = "bbbyyp"
	s["dd"] = "bbbyyp"
	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println("json.Marshal failed:", err)
		return ""
	}

	fmt.Println("map:", string(b))
	return string(b)
}

func main() {
	map2string()
	s := []map[string]interface{}{}

	m1 := map[string]interface{}{"name": "John", "age": 10}
	m2 := map[string]interface{}{"name": "Alex", "age": 12}

	s = append(s, m1, m2)
	s = append(s, m2)

	b, err := json.Marshal(s)
	if err != nil {
		fmt.Println("json.Marshal failed:", err)
		return

	}

	fmt.Println("b:", string(b))
}
