package main

import (
	"time"

	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

// 日志钩子(日志拦截，并重定向)
func NewLfsHook(logName string, rotationTime time.Duration, leastDay uint) log.Hook {
	writer, err := rotatelogs.New(
		// 日志文件
		logName+".%Y_%m_%d_%H-%M-%S",

		// 日志周期(默认每86400秒/一天旋转一次)
		rotatelogs.WithRotationTime(rotationTime),

		// 清除历史 (WithMaxAge和WithRotationCount只能选其一)
		//rotatelogs.WithMaxAge(time.Hour*24*7), //默认每7天清除下日志文件
		rotatelogs.WithRotationCount(leastDay), //只保留最近的N个日志文件
	)
	if err != nil {
		panic(err)
	}
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(false)

	// 可设置按不同level创建不同的文件名
	lfsHook := lfshook.NewHook(lfshook.WriterMap{
		log.DebugLevel: writer,
		log.InfoLevel:  writer,
		log.WarnLevel:  writer,
		log.ErrorLevel: writer,
		log.FatalLevel: writer,
		log.PanicLevel: writer,
	}, &log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		//PrettyPrint:     true,
	})

	return lfsHook
}
