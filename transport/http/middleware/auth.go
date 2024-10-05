package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

const (
	defaultAuthHeader = "token"
)

const (
	ErrAuthUnauthorized  = "unauthorized"
	ErrAuthHeaderMissing = "missing auth header"
)

type AuthConfig struct {
	// Header is the header key to look for the auth value
	Header string
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from auth
	SkippedPathPrefixes []string
	// Validate is a function that takes a gin context and returns the auth value
	// The fields that must be included are: token, userId, userRole
	// struct{} or a type that implements the String() method
	// example: 
	// 
	Validate func(c *gin.Context) (map[any]any, error)
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	if config.Header == "" {
		config.Header = defaultAuthHeader
	}

	return func(c *gin.Context) {
		if config.Validate == nil || skippedPathPrefixes(c, config.SkippedPathPrefixes...) {
			c.Next()
			return
		}

		result, err := config.Validate(c)
		if err != nil {
			log.Errorw("error", errors.Unauthorized(ErrAuthUnauthorized, err.Error()))
			response.GinJSONError(c, errors.Unauthorized(ErrAuthUnauthorized, err.Error()))
			return
		}
		header := result[config.Header]
		if header == nil {
			log.Errorw("error", errors.Unauthorized(ErrAuthHeaderMissing, "token not found"))
			response.GinJSONError(c, errors.Unauthorized(ErrAuthHeaderMissing, "token not found"))
			return
		}

		// with context
		ctx := c.Request.Context()
		for k, v := range result {
			ctx = context.WithValue(ctx, k, v)
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
