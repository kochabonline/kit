package slog

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/kochabonline/kit/log/level"
)

type Slog struct {
	callerWithSkipFrameCount    int
	addCallerWithSkipFrameCount int
	logger                      *slog.Logger
}

type Option func(*Slog)

func WithCallerSkipFrameCount(skipFrameCount int) Option {
	return func(s *Slog) {
		s.callerWithSkipFrameCount = skipFrameCount
	}
}

func WithAddCallerSkipFrameCount(skipFrameCount int) Option {
	return func(s *Slog) {
		s.addCallerWithSkipFrameCount = skipFrameCount
	}
}

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
	var callerWithSkipFrameCount int
	switch {
	case s.callerWithSkipFrameCount != 0:
		callerWithSkipFrameCount = s.callerWithSkipFrameCount
	case s.addCallerWithSkipFrameCount != 0:
		callerWithSkipFrameCount = callerSkipFrameCount() + s.addCallerWithSkipFrameCount
	default:
		callerWithSkipFrameCount = callerSkipFrameCount()
	}

	switch l {
	case level.Debug:
		s.logger.With("caller", caller(callerWithSkipFrameCount)).Debug(msg, args...)
	case level.Info:
		s.logger.With("caller", caller(callerWithSkipFrameCount)).Info(msg, args...)
	case level.Warn:
		s.logger.With("caller", caller(callerWithSkipFrameCount)).Warn(msg, args...)
	case level.Error:
		s.logger.With("caller", caller(callerWithSkipFrameCount)).Error(msg, args...)
	case level.Fatal:
		s.logger.With("caller", caller(callerWithSkipFrameCount)).Log(context.Background(), slog.Level(level.Fatal), msg, args...)
	}
}

func (s *Slog) Log(l level.Level, args ...any) {
	if len(args) == 0 {
		return
	}

	msg, ok := args[0].(string)
	if !ok {
		s.logger.With("caller", caller(callerSkipFrameCount())).Error("invalid log message, first argument must be a string")
		return
	}
	args = args[1:]

	s.log(l, msg, args...)
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
