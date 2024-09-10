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
	global.Log(level.Debug, defaultMsgKey, fmt.Sprint(args...))
}

func Debugf(format string, args ...any) {
	global.Log(level.Debug, defaultMsgKey, fmt.Sprintf(format, args...))
}

func Debugw(keyvals ...any) {
	global.Log(level.Debug, keyvals...)
}

func Info(args ...any) {
	global.Log(level.Info, defaultMsgKey, fmt.Sprint(args...))
}

func Infof(format string, args ...any) {
	global.Log(level.Info, defaultMsgKey, fmt.Sprintf(format, args...))
}

func Infow(keyvals ...any) {
	global.Log(level.Info, keyvals...)
}

func Warn(args ...any) {
	global.Log(level.Warn, defaultMsgKey, fmt.Sprint(args...))
}

func Warnf(format string, args ...any) {
	global.Log(level.Warn, defaultMsgKey, fmt.Sprintf(format, args...))
}

func Warnw(keyvals ...any) {
	global.Log(level.Warn, keyvals...)
}

func Error(args ...any) {
	global.Log(level.Error, defaultMsgKey, fmt.Sprint(args...))
}

func Errorf(format string, args ...any) {
	global.Log(level.Error, defaultMsgKey, fmt.Sprintf(format, args...))
}

func Errorw(keyvals ...any) {
	global.Log(level.Error, keyvals...)
}

func Fatal(args ...any) {
	global.Log(level.Fatal, defaultMsgKey, fmt.Sprint(args...))
	os.Exit(1)
}

func Fatalf(format string, args ...any) {
	global.Log(level.Fatal, defaultMsgKey, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func Fatalw(keyvals ...any) {
	global.Log(level.Fatal, keyvals...)
	os.Exit(1)
}
