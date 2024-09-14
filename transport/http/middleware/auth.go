package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"gorm.io/gorm"
)

const (
	defaultToken = "token"
)

const (
	ErrAuthHeaderMissingReason = "missing auth header"
	ErrAuthHeaderInvalidReason = "invalid auth header"
	ErrAuthTokenMissingReason  = "missing token"
	ErrAuthTokenInvalidReason  = "invalid token"
)

type AuthConfig struct {
	// AuthHeader is the header key to look for the auth value
	AuthHeader string
	// Token is the header key to look for the auth value
	Token string
	// Validate is a function that takes a gin context and returns the auth value
	// The contents in the map will be put into the context
	// The fields that must be included are: userId, header, token
	// struct{} or a type that implements the String() method
	Validate func(c *gin.Context) (map[any]any, error)
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from auth
	SkippedPathPrefixes []string
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	if config.Token == "" {
		config.Token = defaultToken
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
				handleError(c, authHeader, errors.Unauthorized(ErrAuthTokenMissingReason, "token not found"))
				return
			}
			handleError(c, authHeader, errors.Unauthorized(ErrAuthTokenInvalidReason, "error validating token: %v", err))
			return
		}
		header, token := result[config.AuthHeader], result[config.Token]
		if authHeader != header {
			handleError(c, authHeader, errors.Unauthorized(ErrAuthHeaderInvalidReason, "expected %s, got %s", header, authHeader))
			return
		}
		if token == nil {
			handleError(c, authHeader, errors.Unauthorized(ErrAuthTokenMissingReason, "token not found"))
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
