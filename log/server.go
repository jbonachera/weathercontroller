package log

import (
	"fmt"
	"time"
)

var logChan chan Message

func init() {
	logChan = make(chan Message, 50)
	go loop()
}

func loop() {
	Info("log subsystem started")
	run := true
	for run {
		select {
		case msg := <-logChan:
			printLog(msg)
			msg.Close()
		}
	}
}

func printLog(msg Message) {
	fmt.Printf("%s [%s] %s\n", msg.CreationDate().Format(time.RFC3339), Severity(msg.Severity()), msg.Payload())
}
