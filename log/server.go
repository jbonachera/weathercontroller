package log

import (
	"fmt"
	"time"
)

var logChan chan Message
var doneChan chan bool

func init() {
	logChan = make(chan Message, 50)
	doneChan = make(chan bool, 1)
	go loop()
}

func loop() {
	Info("log subsystem started")
	run := true
	for run {
		select {
		case msg, chanOpen := <-logChan:
			if !chanOpen {
				msg, _ := NewMessage(INFO, "log subsystem flushed and closed")
				printLog(msg)
				doneChan <- true
				return
			} else {
				printLog(msg)
				msg.Close()
			}
		}
	}
}

func printLog(msg Message) {
	fmt.Printf("%s [%s] %s\n", msg.CreationDate().Format(time.RFC3339), Severity(msg.Severity()), msg.Payload())
}
