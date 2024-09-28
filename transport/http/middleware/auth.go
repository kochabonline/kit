package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"gorm.io/gorm"
)

const (
	defaultAuthHeader = "token"
)

const (
	ErrAuthHeaderMissingReason = "missing auth header"
	ErrAuthHeaderInvalidReason = "invalid auth header"
)

type AuthConfig struct {
	// AuthHeader is the header key to look for the auth value
	AuthHeader string
	// Validate is a function that takes a gin context and returns the auth value
	// The contents in the map will be put into the context
	// The fields that must be included are: userId, header, token
	// struct{} or a type that implements the String() method
	Validate func(c *gin.Context) (map[any]any, error)
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from auth
	SkippedPathPrefixes []string
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	if config.AuthHeader == "" {
		config.AuthHeader = defaultAuthHeader
	}

	return func(c *gin.Context) {
		if config.Validate == nil || skippedPathPrefixes(c, config.SkippedPathPrefixes...) {
			c.Next()
			return
		}

		authHeader := c.GetHeader(config.AuthHeader)
		if authHeader == "" {
			handleError(c, authHeader, errors.Unauthorized(ErrAuthHeaderMissingReason, "header %s not found", config.AuthHeader))
			return
		}

		result, err := config.Validate(c)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				handleError(c, authHeader, errors.Unauthorized(ErrAuthHeaderMissingReason, "token not found"))
				return
			}
			handleError(c, authHeader, errors.Unauthorized(ErrAuthHeaderInvalidReason, "error validating token: %v", err))
			return
		}
		header := result[config.AuthHeader]
		if header == nil {
			handleError(c, authHeader, errors.Unauthorized(ErrAuthHeaderMissingReason, "token not found"))
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
