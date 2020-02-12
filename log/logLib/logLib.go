package logLib

import (
	"io"
	"log"
	"os"
)

type LogType int

const (
	C_LOG_TRACE   LogType = iota // value --> 0
	C_LOG_INFO                   // value --> 1
	C_LOG_WORNING                // value --> 2
	C_LOG_ERROR                  // value --> 3
)

var (
	logTrace   *log.Logger // 记录所有日志
	logInfo    *log.Logger // 重要的信息
	logWarning *log.Logger // 需要注意的信息
	logError   *log.Logger // 致命错误
)

func init() {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	// logTrace = log.New(ioutil.Discard, "LOG_TRACE:", log.Ltime|log.Lshortfile)
	// logInfo = log.New(os.Stdout, "LOG_INFO:", log.Ltime|log.Lshortfile)
	// logWarning = log.New(os.Stdout, "LOG_WARNING:", log.Ltime|log.Lshortfile)
	// logError = log.New(io.MultiWriter(file, os.Stderr), "LOG_ERROR:", log.Ltime|log.Lshortfile)

	logTrace = log.New(io.MultiWriter(file, os.Stderr), "LOG_TRACE:", log.Ldate|log.Ltime|log.Lshortfile)
	logInfo = log.New(io.MultiWriter(file, os.Stderr), "LOG_INFO:", log.Ldate|log.Ltime|log.Lshortfile)
	logWarning = log.New(io.MultiWriter(file, os.Stderr), "LOG_WARNING:", log.Ldate|log.Ltime|log.Lshortfile)
	logError = log.New(io.MultiWriter(file, os.Stderr), "LOG_ERROR:", log.Ldate|log.Ltime|log.Lshortfile)
}

func LogMessage(logType LogType, msg string) {
	switch logType {
	case C_LOG_TRACE:
		logTrace.Println(msg)
	case C_LOG_INFO:
		logInfo.Println(msg)
	case C_LOG_WORNING:
		logWarning.Println(msg)
	case C_LOG_ERROR:
		logError.Println(msg)
	}
}
