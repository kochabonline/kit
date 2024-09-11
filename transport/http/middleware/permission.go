package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

var (
	ErrPermissionUnauthorized = errors.Forbidden("permission denied", "permission middleware failed to authorize request")
)

type PermissionConfig struct {
	// Param is the param key to look for the id value
	Param string
	// Validate is a function that takes a gin context and returns the auth value
	Validate func(c *gin.Context) (id string, role string, err error)
	// SkippedRoles is a list of roles that should be skipped from permission
	SkippedRoles []string
}

func PermissionWithConfig(config PermissionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Validate == nil || config.Param == "" {
			c.Next()
			return
		}

		id, role, err := config.Validate(c)
		if err != nil {
			log.Errorf("permission failed to validate: %v", err)
			response.GinJSONError(c, ErrPermissionUnauthorized)
			return
		}

		for _, skippedRole := range config.SkippedRoles {
			if role == skippedRole {
				c.Next()
				return
			}
		}

		paramValue := c.Param(config.Param)
		if paramValue != "" && paramValue != id {
			log.Errorf("permission denied access to %s for %s", paramValue, id)
			response.GinJSONError(c, ErrPermissionUnauthorized)
			return
		}

		c.Next()
	}
}
