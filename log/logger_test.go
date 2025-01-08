package log

import (
	"encoding/json"
	"testing"

	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log/slog"
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
	m := &mock{Name: "test", Age: 10}
	h := NewHelper(zerolog.New().With().HelperCaller().Logger())
	h.Debug("test message")
	h.Debug("test message: ", "value")
	h.Error(errors.New(400, "test"))
	h.Info(m)
	h.Info("test message", "key", "value")
}

func TestLogf(t *testing.T) {
	h := NewHelper(zerolog.New().With().HelperCaller().Logger())
	h.Debugf("test message %s %s", "value", "value")
}

func TestLogw(t *testing.T) {
	h := NewHelper(zerolog.New().With().HelperCaller().Logger())
	h.Debugw("key", "value")
	err := errors.New(400, "test")
	h.Errorw("error", err)

	s := NewHelper(slog.New().With().HelperCaller().Logger())
	s.Debugw("test message", "key", "value")
}
