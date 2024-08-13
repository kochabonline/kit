package log

import (
	"fmt"
	"os"
	"sync"

	"github.com/kochabonline/kit/log/level"
	"github.com/kochabonline/kit/log/zerolog"
)

var (
	global = new(glogger)
)

type glogger struct {
	mu sync.Mutex
	Logger
}

func init() {
	global.setLogger(zerolog.New())
}

func (g *glogger) setLogger(logger Logger) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Logger = logger
}

func SetLogger(logger Logger) {
	global.setLogger(logger)
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
