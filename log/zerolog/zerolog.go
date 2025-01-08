package zerolog

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kochabonline/kit/log/level"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

const (
	defaultMissingValue               = "missing value"
	defaultCallerSkipFrameCount       = 4
	defaultHelperCallerSkipFrameCount = 4
	defaultFilterCallerSkipFrameCount = 5
	defaultGlobalCallerSkipFrameCount = 5
)

type Logger struct {
	logger zerolog.Logger
}

type Context struct {
	l *Logger
}

type Option func(*Logger)

func init() {
	zerolog.TimeFieldFormat = time.DateTime
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func SetGlobalLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

func consoleWriter() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}

	return output
}

func New(opts ...Option) *Logger {
	z := &Logger{
		logger: zerolog.New(consoleWriter()).With().Timestamp().Logger(),
	}

	for _, opt := range opts {
		opt(z)
	}

	return z
}

func NewFile(c Config, opts ...Option) *Logger {
	_ = c.initConfig()
	writer := rotate(c)

	z := &Logger{
		logger: zerolog.New(writer).With().Timestamp().Logger(),
	}

	for _, opt := range opts {
		opt(z)
	}

	return z
}

func NewMulti(c Config, opts ...Option) *Logger {
	_ = c.initConfig()
	writer := rotate(c)
	multi := zerolog.MultiLevelWriter(writer, consoleWriter())

	z := &Logger{
		logger: zerolog.New(multi).With().Timestamp().Logger(),
	}

	for _, opt := range opts {
		opt(z)
	}

	return z
}

func (z *Logger) Log(l level.Level, args ...any) {
	if len(args) == 0 {
		return
	}

	var event *zerolog.Event
	switch l {
	case level.Debug:
		event = z.logger.Debug()
	case level.Info:
		event = z.logger.Info()
	case level.Warn:
		event = z.logger.Warn()
	case level.Error:
		event = z.logger.Error().Stack()
	case level.Fatal:
		event = z.logger.Fatal().Stack()
	}

	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		if i+1 < len(args) {
			event = event.Any(key, args[i+1])
		} else {
			event = event.Any(key, defaultMissingValue)
		}
	}

	event.Send()
}

func (z *Logger) With() *Context {
	return &Context{l: z}
}

func (c *Context) Caller() *Context {
	c.l.logger = c.l.logger.With().CallerWithSkipFrameCount(defaultCallerSkipFrameCount).Logger()
	return c
}

func (c *Context) HelperCaller() *Context {
	c.l.logger = c.l.logger.With().CallerWithSkipFrameCount(defaultHelperCallerSkipFrameCount).Logger()
	return c
}

func (c *Context) FilterCaller() *Context {
	c.l.logger = c.l.logger.With().CallerWithSkipFrameCount(defaultFilterCallerSkipFrameCount).Logger()
	return c
}

func (c *Context) GlobalCaller() *Context {
	c.l.logger = c.l.logger.With().CallerWithSkipFrameCount(defaultGlobalCallerSkipFrameCount).Logger()
	return c
}

func (c *Context) CallerSkipFrameCount(count int) *Context {
	c.l.logger = c.l.logger.With().CallerWithSkipFrameCount(count).Logger()
	return c
}

func (c *Context) Logger() *Logger {
	return c.l
}
