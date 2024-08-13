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
	m := mock{Name: "test"}
	h := NewHelper(zerolog.New().With().HelperCaller().Logger())
	h.Debug("test message", "mock", m)
	h.Info("test message", "key", "value")
	f := NewHelper(NewFilter(zerolog.New().With().FilterCaller().Logger(), WithFilterKey("password")))
	f.Info("test message", "password", "12345", "user", "alex")
	SetLogger(zerolog.New().With().Caller().Logger())
	Error("test message", "error", errors.BadRequest("bad request", "").Error())
	s := NewHelper(slog.New().With().Caller().Logger())
	s.Info("test message", "key", "value")
	s.Debug("test message", "mock", m)
}
