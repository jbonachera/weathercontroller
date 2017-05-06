package log

import "testing"

func TestNewMessage(t *testing.T) {
	message, err := NewMessage(1, "log")
	if err != nil {
		t.Error("NewMessage should create a log message")
	}
	if message.Payload() != "log" {
		t.Error("NewMessage should use the given payload")
	}
	if message.Severity() != 1 {
		t.Error("NewMessage should use the given severity")
	}
}
