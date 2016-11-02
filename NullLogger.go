package log

var NullLogger = &NullLog{}

type NullLog struct {}
func (nl *NullLog) Debug(format string, v ...interface{}) {}
func (nl *NullLog) Error(format string, v ...interface{}) {}
func (nl *NullLog) Info(format string, v ...interface{}) {}
