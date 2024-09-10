package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
)

type LoggerConfig struct {
	Logger *log.Helper

	HeaderEnabled    bool
	BodyEnabled      bool
	UserAgentEnabled bool
	Option           LoggerOption
}

type LoggerOption struct {
	Filter func(c *gin.Context) bool
}

func GinLogger() gin.HandlerFunc {
	config := LoggerConfig{
		Logger: log.NewHelper(log.DefaultLogger),
	}

	return GinLoggerWithConfig(config)
}

func GinLoggerWithConfig(config LoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Logger == nil {
			config.Logger = log.NewHelper(log.DefaultLogger)
		}

		start := time.Now()
		c.Next()
		cost := time.Since(start)

		params := make([]any, 0, 10)
		params = append(params,
			"method", c.Request.Method,
			"uri", c.Request.RequestURI,
			"cost", cost.String(),
			"status", c.Writer.Status(),
			"client_ip", c.ClientIP(),
		)

		if requestId := c.Request.Header.Get("X-Request-Id"); requestId != "" {
			params = append(params, "request_id", requestId)
		}
		if config.HeaderEnabled {
			params = append(params, "headers", c.Request.Header)
		}
		if config.BodyEnabled && config.Option.Filter != nil && config.Option.Filter(c) {
			body, _ := c.GetRawData()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			params = append(params, "body", string(body))
		}
		if config.UserAgentEnabled {
			params = append(params, "user_agent", c.Request.UserAgent())
		}

		if len(c.Errors) > 0 {
			params = append(params, "errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		config.Logger.Infow(params...)
	}
}
