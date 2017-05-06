package log

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

const (
	TRACE = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

var severities = [...]string{
	"TRACE",
	"DEBUG",
	"INFO ",
	"WARN ",
	"ERROR",
	"FATAL",
}

func Severity(severity int) string {
	return severities[severity]
}

type Message interface {
	Uuid() uuid.UUID
	Payload() string
	Severity() int
	CreationDate() time.Time
	Close()
}

type message struct {
	uuid         uuid.UUID
	creationDate time.Time
	payload      string
	severity     int
}

func NewMessage(severity int, payload string) (Message, error) {
	if severity > FATAL {
		return nil, errors.New("unknown severity")
	} else {
		return &message{uuid: uuid.New(), creationDate: time.Now(), payload: payload, severity: severity}, nil
	}
}
func (message *message) CreationDate() time.Time {
	return message.creationDate
}
func (message *message) Payload() string {
	return message.payload
}
func (message *message) Uuid() uuid.UUID {
	return message.uuid
}
func (message *message) Severity() int {
	return message.severity
}
func (message *message) Close() {
	// Use this to send metrics about message consumption time,
	// but do NOT produce a log, as this is called by the log routine
}
