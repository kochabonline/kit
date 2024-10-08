package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/core/tools"
)

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

func findRoleWithEmpty(userRole string, roles ...string) bool {
	if len(roles) == 0 {
		return true
	}

	for _, role := range roles {
		if userRole == role {
			return true
		}
	}

	return false
}

func userInfo(c *gin.Context) (userId int64, userRole string, err error) {
	ctx := c.Request.Context()
	userId, err = tools.CtxValue[int64](ctx, "userId")
	if err != nil {
		return
	}
	userRole, err = tools.CtxValue[string](ctx, "userRole")
	if err != nil {
		return
	}
	return
}
