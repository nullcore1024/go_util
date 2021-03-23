package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-yaml/yaml"
)

type Visitor func(person Person)

// 基类
// 被访问的对象，通过accept方法接受访问
type Person interface {
	accept(visitor Visitor)
}

// 存储学生信息的类型
// 实现了Person接口
type Student struct {
	Name  string `yaml:name`
	Age   int
	Score int
}

func (s Student) accept(visitor Visitor) {
	visitor(s)
}

// 存储教师信息
type Teacher struct {
	Name   string
	Age    int
	Course string
}

func (t Teacher) accept(visitor Visitor) {
	visitor(t)
}

func JsonVisitor(person Person) {
	bytes, err := json.Marshal(person)
	if err != nil {
		panic(err)
	}
	fmt.Println("json:", string(bytes))
}

// 导出yaml格式信息的访问器
func YamlVisitor(person Person) {
	bytes, err := yaml.Marshal(person)
	if err != nil {
		panic(err)
	}
	fmt.Println("yaml:", string(bytes))
}

func main() {
	s := Student{Age: 10, Name: "小明", Score: 90}
	t := Teacher{Name: "李", Age: 35, Course: "数学"}
	persons := []Person{s, t}

	for _, person := range persons {
		person.accept(JsonVisitor)
		fmt.Println("==")
		person.accept(YamlVisitor)
	}
}
