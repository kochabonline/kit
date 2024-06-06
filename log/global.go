package log

import (
	"fmt"
	"os"
	"sync"

	"github.com/kochabonline/kit/log/level"
	"github.com/kochabonline/kit/log/zerolog"
)

var global = new(glogger)

type glogger struct {
	mu sync.Mutex
	Logger
}

func init() {
	global.SetLogger(zerolog.New(zerolog.DefaultSkipFrameCount))
}

func SetDefaultLogger(logger Logger) {
	global.SetLogger(logger)
}

func (l *glogger) SetLogger(logger Logger) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Logger = logger
}

func Debug(args ...any) {
	global.Log(level.Debug, args...)
}

func Debugf(format string, args ...any) {
	global.Log(level.Debug, fmt.Sprintf(format, args...))
}

func Info(args ...any) {
	global.Log(level.Info, args...)
}

func Infof(format string, args ...any) {
	global.Log(level.Info, fmt.Sprintf(format, args...))
}

func Warn(args ...any) {
	global.Log(level.Warn, args...)
}

func Warnf(format string, args ...any) {
	global.Log(level.Warn, fmt.Sprintf(format, args...))
}

func Error(args ...any) {
	global.Log(level.Error, args...)
}

func Errorf(format string, args ...any) {
	global.Log(level.Error, fmt.Sprintf(format, args...))
}

func Fatal(args ...any) {
	global.Log(level.Fatal, args...)
	os.Exit(1)
}

func Fatalf(format string, args ...any) {
	global.Log(level.Fatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}
