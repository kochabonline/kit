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
	defaultAuthHeader = "token"
)

const (
	ErrAuthHeaderMissing = "missing auth header"
	ErrAuthHeaderInvalid = "invalid auth header"
)

type AuthConfig struct {
	// Header is the header key to look for the auth value
	Header string
	// Validate is a function that takes a gin context and returns the auth value
	// The contents in the map will be put into the context
	// The fields that must be included are: userId, token
	// struct{} or a type that implements the String() method
	Validate func(c *gin.Context) (map[any]any, error)
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from auth
	SkippedPathPrefixes []string
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
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Errorw("error", errors.Unauthorized(ErrAuthHeaderMissing, "token not found"))
				response.GinJSONError(c, errors.Unauthorized(ErrAuthHeaderMissing, "token not found"))
				return
			}
			log.Errorw("error", errors.Unauthorized(ErrAuthHeaderInvalid, "error validating token: %v", err))
			response.GinJSONError(c, errors.Unauthorized(ErrAuthHeaderInvalid, "error validating token: %v", err))
			return
		}
		header := result[config.Header]
		if header == nil {
			log.Errorw("error", errors.Unauthorized(ErrAuthHeaderMissing, "token not found"), config.Header, header)
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
