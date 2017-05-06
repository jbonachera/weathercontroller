package log

import "fmt"

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
	fmt.Printf("[%s] %s\n", Severity(msg.Severity()), msg.Payload())
}
