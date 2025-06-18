package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

type LoggerConfig struct {
	// Enable logging of request headers
	HeaderEnabled  bool
	BodyEnabled    bool
	HandlerEnabled bool
	// Option to filter requests to be logged
	Option LoggerOption
}

type LoggerOption struct {
	Filter func(c *gin.Context)
}

func GinLogger() gin.HandlerFunc {
	return GinLoggerWithConfig(LoggerConfig{})
}

func GinLoggerWithConfig(config LoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Option.Filter != nil {
			config.Option.Filter(c)
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		event := mlog.Info().
			Str("method", c.Request.Method).
			Str("uri", c.Request.RequestURI).
			Dur("duration", duration).
			Int("status", c.Writer.Status()).
			Str("client_ip", c.ClientIP())

		if config.BodyEnabled {
			body, err := c.GetRawData()
			if err != nil {
				event = event.Str("error", err.Error())
			} else {
				event = event.Str("body", string(body))
			}
		}

		if requestId := c.Request.Header.Get("X-Request-Id"); requestId != "" {
			event = event.Str("request_id", requestId)
		}

		if config.HeaderEnabled {
			event = event.Any("headers", c.Request.Header)
		}

		if config.HandlerEnabled {
			event = event.Str("handler", c.HandlerName())
		}

		if len(c.Errors) > 0 {
			event = event.Any("errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		event.Send()
	}
}
