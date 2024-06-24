package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

var ErrUnauthorized = errors.Unauthorized("unauthorized", "unauthorized")

type AuthConfig struct {
	AuthHeader string
	ParseId    func(c *gin.Context) (int64, error)
}

func AuthWithConfig(config AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		headerUserId, err := parseHeaderId(c, config.AuthHeader)
		if err != nil {
			log.Error("failed to parse header id", "error", err)
			response.GinJSONError(c, err)
			return
		}

		userId, err := config.ParseId(c)
		if err != nil {
			log.Error("failed to parse id", "error", err)
			response.GinJSONError(c, err)
			return
		}

		if headerUserId != userId {
			log.Error("unauthorized", "headerUserId", headerUserId, "userId", userId)
			response.GinJSONError(c, ErrUnauthorized)
			return
		}

		c.Next()
	}
}

func parseHeaderId(c *gin.Context, header string) (int64, error) {
	id := c.GetHeader(header)
	if id == "" {
		return 0, errors.BadRequest("missing header", "missing header %s", header)
	}

	parse, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, errors.BadRequest("invalid header", "invalid header %s", id)
	}

	return parse, nil
}
