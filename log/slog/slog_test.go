package slog

import (
	"encoding/json"
	"testing"

	"github.com/kochabonline/kit/log/level"
)

type mock struct {
	Name string
	Age  int
}

func (m mock) String() string {
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

func TestSlog(t *testing.T) {
	logger := New()

	logger.Log(level.Debug, "debug message", "user", mock{Name: "John", Age: 30})
	logger.Log(level.Info, "info message")
	logger.Log(level.Warn, "warn message")
	logger.Log(level.Error, "error message")
	logger.Log(level.Fatal, "fatal message")
}
