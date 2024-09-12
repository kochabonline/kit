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
	ErrAuthHeaderNotFound = errors.BadRequest("missing header", "auth middleware failed to authorize request")
	ErrAuthNotLoggedIn    = errors.BadRequest("not logged in", "auth middleware failed to authorize request")
	ErrAuthUnauthorized   = errors.Unauthorized("unauthorized", "auth middleware failed to authorize request")
	ErrAuthForbidden      = errors.Forbidden("permission denied", "auth middleware failed to authorize request")
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
	return func(c *gin.Context) {
		if config.Validate == nil || skippedPathPrefixes(c, config.SkippedPathPrefixes...) {
			c.Next()
			return
		}
		if config.Token == "" {
			config.Token = defaultToken
		}

		authHeader := c.GetHeader(config.AuthHeader)
		if authHeader == "" {
			log.Errorw("user", authHeader, "error", ErrAuthHeaderNotFound)
			response.GinJSONError(c, ErrAuthHeaderNotFound)
			return
		}

		result, err := config.Validate(c)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Errorw("user", authHeader, "error", ErrAuthUnauthorized)
				response.GinJSONError(c, ErrAuthUnauthorized)
				return
			}
			log.Errorw("user", authHeader, "error", err)
			response.GinJSONError(c, ErrAuthUnauthorized)
			return
		}
		header, token := result[config.AuthHeader], result[config.Token]
		if authHeader != header {
			log.Errorw("user", authHeader, "error", ErrAuthUnauthorized)
			response.GinJSONError(c, ErrAuthUnauthorized)
			return
		}
		if token == nil {
			log.Errorw("user", header, "error", ErrAuthNotLoggedIn)
			response.GinJSONError(c, ErrAuthNotLoggedIn)
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
