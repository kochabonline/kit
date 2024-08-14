package slog

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kochabonline/kit/log/level"
)

const (
	defaultCallerSkipFrameCount       = 4
	defaultHelperCallerSkipFrameCount = 4
	defaultFilterCallerSkipFrameCount = 5
	defaultGlobalCallerSkipFrameCount = 5
)

type Slog struct {
	caller               bool
	callerSkipFrameCount int
	logger               *slog.Logger
}

type Context struct {
	l *Slog
}

type Option func(*Slog)

func New(opts ...Option) *Slog {
	s := &Slog{
		logger: slog.New(slog.NewTextHandler(os.Stdout, HandlerOptions())),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Slog) log(l level.Level, msg string, args ...any) {
	if s.caller {
		args = append(args, "caller", caller(s.callerSkipFrameCount))
	}

	switch l {
	case level.Debug:
		s.logger.Debug(msg, args...)
	case level.Info:
		s.logger.Info(msg, args...)
	case level.Warn:
		s.logger.Warn(msg, args...)
	case level.Error:
		s.logger.Error(msg, args...)
	case level.Fatal:
		s.logger.Log(context.Background(), slog.Level(level.Fatal), msg, args...)
	}
}

func (s *Slog) Log(l level.Level, args ...any) {
	if len(args) == 0 {
		return
	}

	msg, ok := args[0].(string)
	if !ok {
		s.logger.Error("first argument must be a string")
		return
	}
	args = args[1:]

	s.log(l, msg, args...)
}

func (s *Slog) With() *Context {
	return &Context{l: s}
}

func (c *Context) Caller() *Context {
	c.l.caller = true
	c.l.callerSkipFrameCount = defaultCallerSkipFrameCount
	return c
}

func (c *Context) HelperCaller() *Context {
	c.l.caller = true
	c.l.callerSkipFrameCount = defaultHelperCallerSkipFrameCount
	return c
}

func (c *Context) FilterCaller() *Context {
	c.l.caller = true
	c.l.callerSkipFrameCount = defaultFilterCallerSkipFrameCount
	return c
}

func (c *Context) GlobalCaller() *Context {
	c.l.caller = true
	c.l.callerSkipFrameCount = defaultGlobalCallerSkipFrameCount
	return c
}

func (c *Context) CallerSkipFrameCount(count int) *Context {
	c.l.caller = true
	c.l.callerSkipFrameCount = count
	return c
}

func (c *Context) Logger() *Slog {
	return c.l
}

func HandlerOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		Level: slog.Level(level.Debug),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.DateTime))
			}

			if a.Key == slog.LevelKey {
				levelLabel := a.Value.Any().(slog.Level)

				switch {
				case levelLabel < slog.Level(level.Debug):
					a.Value = slog.StringValue("TRACE")
				case levelLabel < slog.Level(level.Info):
					a.Value = slog.StringValue("DEBUG")
				case levelLabel < slog.Level(level.Warn):
					a.Value = slog.StringValue("INFO")
				case levelLabel < slog.Level(level.Error):
					a.Value = slog.StringValue("WARN")
				case levelLabel < slog.Level(level.Fatal):
					a.Value = slog.StringValue("ERROR")
				default:
					a.Value = slog.StringValue("FATAL")
				}
			}

			return a
		},
	}
}

func caller(depth int) string {
	_, file, line, _ := runtime.Caller(depth)
	idx := strings.LastIndexByte(file, '/')
	if idx == -1 {
		return file[idx+1:] + ":" + strconv.Itoa(line)
	}
	idx = strings.LastIndexByte(file[:idx], '/')
	return file[idx+1:] + ":" + strconv.Itoa(line)
}
