package log

import (
	"github.com/kochabonline/kit/log/level"
	"github.com/kochabonline/kit/log/zerolog"
)

var (
	DefaultLogger = NewHelper(zerolog.New(zerolog.DefaultSkipFrameCount))
)

type Logger interface {
	Log(level level.Level, keyvals ...any)
}
