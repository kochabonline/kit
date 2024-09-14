package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

func handleError(c *gin.Context, user string, err error) {
	if err == nil {
		return
	}

	log.Errorw("user", user, "error", err)
	response.GinJSONError(c, err)
}

func skippedPathPrefixes(c *gin.Context, prefixes ...string) bool {
	if len(prefixes) == 0 {
		return false
	}

	path := c.Request.URL.Path
	for _, prefix := range prefixes {
		if path == prefix || len(path) > len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}
