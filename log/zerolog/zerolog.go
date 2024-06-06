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
	defaultMsgKey = "msg"
)

type Logger struct {
	logger zerolog.Logger
	once   bool
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
	if !z.once {
		z.logger = z.logger.With().CallerWithSkipFrameCount(callerSkipFrameCount()).Logger()
		z.once = true
	}
	var event *zerolog.Event

	length := len(args)
	if length == 0 {
		return
	}

	switch l {
	case level.Debug:
		event = z.logger.Debug()
	case level.Info:
		event = z.logger.Info().Stack()
	case level.Warn:
		event = z.logger.Warn()
	case level.Error:
		event = z.logger.Error().Stack()
	case level.Fatal:
		event = z.logger.Fatal().Stack()
	}

	event = event.Any(defaultMsgKey, args[0])
	args = args[1:]
	length--

	for i := 0; i < length; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		if i == length-1 {
			event = event.Any(key, "missing value")
		} else {
			event = event.Any(key, args[i+1])
		}
	}

	event.Send()
}
