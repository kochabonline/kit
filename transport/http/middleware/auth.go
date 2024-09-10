package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

type contextKey string

const (
	Token = contextKey("token")
)

var ErrUnauthorized = errors.Unauthorized("unauthorized", "auth middleware failed to authorize request")

type AuthConfig struct {
	// AuthHeader is the header key to look for the auth value
	AuthHeader string
	// Validate is a function that takes a gin context and returns the auth value
	Validate func(c *gin.Context) (any, string, error)
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from auth
	SkippedPathPrefixes []string
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Validate == nil {
			c.Next()
			return
		}
		if skippedPathPrefixes(c, config.SkippedPathPrefixes...) {
			c.Next()
			return
		}

		authHeader := c.GetHeader(config.AuthHeader)
		if authHeader == "" {
			log.Errorf("unauthorized: missing auth header: %s", config.AuthHeader)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		token, header, err := config.Validate(c)
		if err != nil {
			log.Errorf("unauthorized: failed to validate auth: %v", err)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}
		if authHeader != header {
			log.Errorf("unauthorized: not match header, expected: %s, got: %s", header, authHeader)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}
		if token == nil {
			log.Errorf("unauthorized: token is nil")
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		// Set the token in the context
		ctx := context.WithValue(c.Request.Context(), Token, token)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
