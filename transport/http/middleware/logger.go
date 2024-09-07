package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
)

type GinLoggerConfig struct {
	Logger *log.Helper
}

func GinLogger() gin.HandlerFunc {
	return GinLoggerWithConfig(GinLoggerConfig{
		Logger: log.NewHelper(zerolog.New(zerolog.WithMsgKey(""))),
	})
}

func GinLoggerWithConfig(config GinLoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if len(c.Errors) > 0 {
			params = append(params, "errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		config.Logger.Info(params...)
	}
}
