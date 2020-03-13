package main

import (
	"./logs"
	"fmt"
	"github.com/sirupsen/logrus"
)

func main() {
	//创建一个hook，将日志存储路径输入进去
	hook := logs.NewHook("log.log")
	//加载hook之前打印日志
	logrus.WithField("file", "d:/log/golog.log").Info("New logrus hook err.")
	logrus.AddHook(hook)
	//加载hook之后打印日志
	logrus.WithFields(logrus.Fields{
		"animal": "walrus",
	}).Info("A walrus appears")
}
