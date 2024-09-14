package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

var (
	ErrPermissionForbiddenReason = "permission denied"
)

type PermissionConfig struct {
	// Param is the param key to look for the id value
	Param string
	// Validate is a function that takes a gin context and returns the auth value
	Validate func(c *gin.Context) (id int64, role int, err error)
	// SkippedRoles is a list of roles that should be skipped from permission
	SkippedRole int
}

func PermissionWithConfig(config PermissionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Validate == nil || config.Param == "" {
			c.Next()
			return
		}

		paramValue := c.Param(config.Param)
		if paramValue == "" {
			c.Next()
			return
		}

		id, role, err := config.Validate(c)
		if err != nil {
			handleError(c, strconv.FormatInt(id, 10), errors.Forbidden(ErrPermissionForbiddenReason, "permission failed to validate: %v", err))
			return
		}

		if role >= config.SkippedRole {
			c.Next()
			return
		}

		idStr := strconv.FormatInt(id, 10)
		if paramValue != idStr {
			handleError(c, idStr, errors.Forbidden(ErrPermissionForbiddenReason, "%s is not allowed to access %s", idStr, paramValue))
			return
		}

		c.Next()
	}
}
