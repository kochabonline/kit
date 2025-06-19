package log

import (
	"os"

	"github.com/rs/zerolog"
)

var (
	global *Logger
)

func init() {
	global = New()
}

// SetGlobalLogger 设置全局日志记录器
func SetGlobalLogger(logger *Logger) {
	global = logger
}

// Debug 全局debug日志
func Debug() *zerolog.Event {
	return global.Debug()
}

// Info 全局info日志
func Info() *zerolog.Event {
	return global.Info()
}

// Warn 全局warn日志
func Warn() *zerolog.Event {
	return global.Warn()
}

// Error 全局error日志
func Error() *zerolog.Event {
	return global.Error().Stack()
}

// Fatal 全局fatal日志
func Fatal() *zerolog.Event {
	return global.Fatal().Stack()
}

// Panic 全局panic日志
func Panic() *zerolog.Event {
	return global.Panic().Stack()
}

func Debugf(format string, args ...any) {
	global.Debug().Msgf(format, args...)
}

func Infof(format string, args ...any) {
	global.Info().Msgf(format, args...)
}

func Warnf(format string, args ...any) {
	global.Warn().Msgf(format, args...)
}

func Errorf(format string, args ...any) {
	global.Error().Stack().Msgf(format, args...)
}

func Fatalf(format string, args ...any) {
	global.Fatal().Stack().Msgf(format, args...)
	os.Exit(1)
}

func Panicf(format string, args ...any) {
	global.Panic().Stack().Msgf(format, args...)
}

func Debugs(msg string) {
	global.Debug().Msg(msg)
}

func Infos(msg string) {
	global.Info().Msg(msg)
}

func Warns(msg string) {
	global.Warn().Msg(msg)
}

func Errors(msg string) {
	global.Error().Stack().Msg(msg)
}

func Fatals(msg string) {
	global.Fatal().Stack().Msg(msg)
	os.Exit(1)
}

func Panics(msg string) {
	global.Panic().Stack().Msg(msg)
}
