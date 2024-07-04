package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

var ErrUnauthorized = errors.Unauthorized("unauthorized", "auth middleware failed to authorize request")

type AuthConfig struct {
	// AuthHeader is the header key to look for the auth value
	AuthHeader string
	// Validate is a function that takes a gin context and returns the auth value
	Validate func(c *gin.Context) (string, error)
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from auth
	SkippedPathPrefixes []string
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if skippedPathPrefixes(c, config.SkippedPathPrefixes) {
			c.Next()
			return
		}
		if config.Validate == nil {
			c.Next()
			return
		}

		authHeader := c.GetHeader(config.AuthHeader)
		if authHeader == "" {
			log.Errorf("missing auth header %s", config.AuthHeader)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		auth, err := config.Validate(c)
		if err != nil {
			log.Errorf("auth validation failed: %v", err)
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
