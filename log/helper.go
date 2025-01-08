package log

import (
	"fmt"
	"os"

	"github.com/kochabonline/kit/log/level"
)

const defaultMsgKey = "msg"

type Helper struct {
	logger  Logger
	msgKey  string
	sprint  func(...any) string
	sprintf func(string, ...any) string
}

type HelperOption func(*Helper)

func WithMsgKey(key string) HelperOption {
	return func(h *Helper) {
		h.msgKey = key
	}
}

func WithSprint(sprint func(...any) string) HelperOption {
	return func(h *Helper) {
		h.sprint = sprint
	}
}

func WithSprintf(sprintf func(string, ...any) string) HelperOption {
	return func(h *Helper) {
		h.sprintf = sprintf
	}
}

func NewHelper(logger Logger, opts ...HelperOption) Helper {
	h := Helper{
		logger: logger,
		msgKey: defaultMsgKey,
	}

	for _, opt := range opts {
		opt(&h)
	}

	if h.sprint == nil {
		h.sprint = fmt.Sprint
	}

	if h.sprintf == nil {
		h.sprintf = fmt.Sprintf
	}

	return h
}

func (h *Helper) Debug(args ...any) {
	h.logger.Log(level.Debug, h.msgKey, h.sprint(args...))
}

func (h *Helper) Debugf(format string, args ...any) {
	h.logger.Log(level.Debug, h.msgKey, h.sprintf(format, args...))
}

func (h *Helper) Debugw(keyvals ...any) {
	h.logger.Log(level.Debug, keyvals...)
}

func (h *Helper) Info(args ...any) {
	h.logger.Log(level.Info, h.msgKey, h.sprint(args...))
}

func (h *Helper) Infof(format string, args ...any) {
	h.logger.Log(level.Info, h.msgKey, h.sprintf(format, args...))
}

func (h *Helper) Infow(keyvals ...any) {
	h.logger.Log(level.Info, keyvals...)
}

func (h *Helper) Warn(args ...any) {
	h.logger.Log(level.Warn, h.msgKey, h.sprint(args...))
}

func (h *Helper) Warnf(format string, args ...any) {
	h.logger.Log(level.Warn, h.msgKey, h.sprintf(format, args...))
}

func (h *Helper) Warnw(keyvals ...any) {
	h.logger.Log(level.Warn, keyvals...)
}

func (h *Helper) Error(args ...any) {
	h.logger.Log(level.Error, h.msgKey, h.sprint(args...))
}

func (h *Helper) Errorf(format string, args ...any) {
	h.logger.Log(level.Error, h.msgKey, h.sprintf(format, args...))
}

func (h *Helper) Errorw(keyvals ...any) {
	h.logger.Log(level.Error, keyvals...)
}

func (h *Helper) Fatal(args ...any) {
	h.logger.Log(level.Fatal, h.msgKey, h.sprint(args...))
	os.Exit(1)
}

func (h *Helper) Fatalf(format string, args ...any) {
	h.logger.Log(level.Fatal, h.msgKey, h.sprintf(format, args...))
	os.Exit(1)
}

func (h *Helper) Fatalw(keyvals ...any) {
	h.logger.Log(level.Fatal, keyvals...)
	os.Exit(1)
}
