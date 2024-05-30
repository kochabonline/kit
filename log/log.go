package log

import "github.com/kochabonline/kit/log/level"

type Logger interface {
	Log(level level.Level, keyvals ...any)
}
