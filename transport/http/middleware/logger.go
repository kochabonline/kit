package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
)

type GinLoggerConfig struct {
	Logger *log.Helper
}

func GinLogger() gin.HandlerFunc {
	return GinLoggerWithConfig(GinLoggerConfig{
		Logger: log.DefaultLogger,
	})
}

func GinLoggerWithConfig(config GinLoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		cost := time.Since(start)

		params := make([]any, 0, 12)
		params = append(params,
			"timestamp", time.Now().Format(time.RFC3339),
			"method", c.Request.Method,
			"uri", c.Request.RequestURI,
			"cost", cost.String(),
			"status", c.Writer.Status(),
			"client_ip", c.ClientIP(),
			"query", c.Request.URL.Query(),
			"header", c.Request.Header,
			"body", c.Request.Body,
			"user_agent", c.Request.UserAgent(),
		)

		if requestId := c.Writer.Header().Get("X-Request-ID"); requestId != "" {
			params = append(params, "request_id", requestId)
		}

		if len(c.Errors) > 0 {
			params = append(params, "errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		config.Logger.Infow(params...)
	}
}
