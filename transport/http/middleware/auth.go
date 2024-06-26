package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

var ErrUnauthorized = errors.Unauthorized("unauthorized", "auth middleware failed to authorize request")

type AuthConfig struct {
	AuthHeader string
	ParseAuth  func(ctx context.Context) (string, error)
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.ParseAuth == nil {
			c.Next()
			return
		}

		authHeader := c.GetHeader(config.AuthHeader)
		if authHeader == "" {
			log.Errorf("missing auth header %s", config.AuthHeader)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		auth, err := config.ParseAuth(c.Request.Context())
		if err != nil {
			log.Error("failed to parse auth", "error", err)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		if authHeader != auth {
			log.Error("unauthorized", "authHeader", authHeader, "auth", auth)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		c.Next()
	}
}
