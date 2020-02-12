package main

import (
	"./logLib"
)

func main() {
	logLib.LogMessage(logLib.C_LOG_TRACE, "I have something standard to say")
	logLib.LogMessage(logLib.C_LOG_INFO, "Special Information")
	logLib.LogMessage(logLib.C_LOG_WORNING, "There is something you need to know about")
	logLib.LogMessage(logLib.C_LOG_ERROR, "Something has failed")
}
