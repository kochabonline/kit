package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
	"gorm.io/gorm"
)

const (
	defaultToken = "token"
)

var (
	ErrAuthHeaderNotFound  = errors.Unauthorized("missing auth header", "auth header not found")
	ErrorAuthHeaderInvalid = errors.Unauthorized("invalid auth header", "auth header is invalid")
	ErrAuthTokenNotFound   = errors.Unauthorized("missing token", "token not found")
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
			log.Errorw("error", errors.Unauthorized("missing auth header", "header %s not found", config.AuthHeader))
			response.GinJSONError(c, ErrAuthHeaderNotFound)
			return
		}

		result, err := config.Validate(c)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Errorw("user", authHeader, "error", ErrAuthTokenNotFound)
				response.GinJSONError(c, ErrAuthTokenNotFound)
				return
			}
			log.Errorw("user", authHeader, "error", err)
			response.GinJSONError(c, err)
			return
		}
		header, token := result[config.AuthHeader], result[config.Token]
		if authHeader != header {
			log.Errorw("user", authHeader, "error", errors.Unauthorized("invalid auth header", "expected %s, got %s", header, authHeader))
			response.GinJSONError(c, ErrorAuthHeaderInvalid)
			return
		}
		if token == nil {
			log.Errorw("user", authHeader, "error", ErrAuthTokenNotFound)
			response.GinJSONError(c, ErrAuthTokenNotFound)
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
