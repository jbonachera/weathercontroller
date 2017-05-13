package log

import "fmt"

var logLevel int = INFO
var closed bool = false

func publish(severity int, a ...interface{}) {
	if !closed {
		if severity >= logLevel {
			msg, err := NewMessage(severity, fmt.Sprint(a...))
			if err == nil {
				logChan <- msg
			} else {
				fmt.Println(err)
			}
		}
	}
}

func Info(a ...interface{}) {
	publish(INFO, a...)
}

func Warn(a ...interface{}) {
	publish(WARN, a...)
}

func Fatal(a ...interface{}) {
	publish(FATAL, a...)
}

func Debug(a ...interface{}) {
	publish(DEBUG, a...)
}

func Error(a ...interface{}) {
	publish(ERROR, a...)
}

func Trace(a ...interface{}) {
	publish(TRACE, a...)
}

func SetLevel(severity int) {
	if severity <= FATAL {
		logLevel = severity
	}
}

func Flush() {
	Debug("flushing logs...")
	closed = true
	close(logChan)
	<-doneChan
}
