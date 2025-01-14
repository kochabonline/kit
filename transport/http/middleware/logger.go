package middleware

import (
	"bytes"
	"io"
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

		// Pre-allocate slices to avoid dynamic expansion
		params := make([]any, 0, 10)
		// Reading the request body must be done before c.Next(), otherwise the body will be cleared
		if config.BodyEnabled {
			// After reading, write it back to the request body
			body, _ := c.GetRawData()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			params = append(params, "body", string(body))
		}

		start := time.Now()
		c.Next()
		cost := time.Since(start)

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
		if config.HandlerEnabled {
			params = append(params, "handler", c.HandlerName())
		}

		if len(c.Errors) > 0 {
			params = append(params, "errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		mlog.Infow(params...)
	}
}
