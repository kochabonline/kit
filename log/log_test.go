package log

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/kochabonline/kit/log/zerolog"
)

type mock struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (m *mock) String() string {
	bytes, _ := json.Marshal(m)

	return string(bytes)
}

func TestLog(t *testing.T) {
	m := mock{Name: "test"}
	h := NewHelper(zerolog.New())
	h.Debug("test message", "key", "value")
	h.Debug("test message", "mock", m)
	h.Info("test message", "key", "value")
	f := NewHelper(NewFilter(zerolog.New(), WithFilterKey("password")))
	f.Info("test message", "password", "12345", "user", "alex")
	Error("test message", "error", errors.New("error message").Error())
}
