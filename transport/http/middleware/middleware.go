package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func skippedPathPrefixes(c *gin.Context, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}

	path := c.Request.URL.Path
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}