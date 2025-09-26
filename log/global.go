package log

import (
	"github.com/rs/zerolog"
)

var (
	L *Logger
)

func init() {
	L = New()
}

// SetGlobalLogger 设置全局日志记录器
func SetGlobalLogger(logger *Logger) {
	L = logger
}

// SetGlobalLevel 设置全局日志级别
func SetGlobalLevel(level zerolog.Level) {
	L.Logger = L.Logger.Level(level)
}

// Debug 全局debug日志
func Debug() *zerolog.Event {
	return L.Debug()
}

// Info 全局info日志
func Info() *zerolog.Event {
	return L.Info()
}

// Warn 全局warn日志
func Warn() *zerolog.Event {
	return L.Warn()
}

// Error 全局error日志
func Error() *zerolog.Event {
	return L.Error().Stack()
}

// Fatal 全局fatal日志
func Fatal() *zerolog.Event {
	return L.Fatal().Stack()
}

// Panic 全局panic日志
func Panic() *zerolog.Event {
	return L.Panic().Stack()
}

func Debugf(format string, args ...any) {
	L.Debug().Msgf(format, args...)
}

func Infof(format string, args ...any) {
	L.Info().Msgf(format, args...)
}

func Warnf(format string, args ...any) {
	L.Warn().Msgf(format, args...)
}

func Errorf(format string, args ...any) {
	L.Error().Stack().Msgf(format, args...)
}

func Fatalf(format string, args ...any) {
	L.Fatal().Stack().Msgf(format, args...)
}

func Panicf(format string, args ...any) {
	L.Panic().Stack().Msgf(format, args...)
}

func Debugs(msg string) {
	L.Debug().Msg(msg)
}

func Infos(msg string) {
	L.Info().Msg(msg)
}

func Warns(msg string) {
	L.Warn().Msg(msg)
}

func Errors(msg string) {
	L.Error().Stack().Msg(msg)
}

func Fatals(msg string) {
	L.Fatal().Stack().Msg(msg)
}

func Panics(msg string) {
	L.Panic().Stack().Msg(msg)
}
