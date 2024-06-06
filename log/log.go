package log

import (
	"github.com/kochabonline/kit/log/level"
	"github.com/kochabonline/kit/log/zerolog"
)

var (
	DefaultLogger = NewHelper(zerolog.New())
)

type Logger interface {
	Log(level level.Level, keyvals ...any)
}
