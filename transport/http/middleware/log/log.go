package log

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
)

type Log struct {
	logger *log.Helper
}

type Option func(*Log)

func WithLogger(logger *log.Helper) Option {
	return func(l *Log) {
		l.logger = logger
	}
}

func NewLog(opts ...Option) *Log {
	l := &Log{
		logger: log.NewHelper(zerolog.New()),
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func (l *Log) GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		cost := time.Since(start).String()

		params := l.parseParams(c, cost)

		if len(c.Errors) > 0 {
			params = append(params, "errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		l.logger.Info("http request", params)
	}
}

func (l *Log) parseParams(c *gin.Context, cost string) []any {
	return []any{
		"method", c.Request.Method,
		"uri", c.Request.RequestURI,
		"cost", cost,
		"status", c.Writer.Status(),
		"client_ip", c.ClientIP(),
	}
}
