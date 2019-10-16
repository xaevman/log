package log

import (
	"fmt"
)

var ConsoleLogger = &ConsoleLog{}

type ConsoleLog struct{}

func (nl *ConsoleLog) Debug(format string, v ...interface{}) {
	nl.Print(NewLogMsg("debug", format, 2, v...))
}

func (nl *ConsoleLog) Error(format string, v ...interface{}) {
	nl.Print(NewLogMsg("error", format, 2, v...))
}

func (nl *ConsoleLog) Info(format string, v ...interface{}) {
	nl.Print(NewLogMsg("info", format, 2, v...))
}

func (nl *ConsoleLog) Print(msg *LogMsg) {
	fmt.Print(msg)
}
