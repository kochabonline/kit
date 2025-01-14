package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/transport/http/response"
)

type PermissionHPEConfig struct {
	SkippedRole int
}

func PermissionHPE() gin.HandlerFunc {
	return PermissionHPEWithConfig(PermissionHPEConfig{})
}

func PermissionHPEWithConfig(config PermissionHPEConfig) gin.HandlerFunc {

	return func(c *gin.Context) {
		_, accountRole, err := ctxAccountInfo(c)
		if err != nil {
			mlog.Errorw("error", err.Error())
			response.GinJSONError(c, ErrorUnauthorized)
			return
		}

		if config.SkippedRole >= accountRole {
			c.Next()
			return
		}

		c.Next()
	}
}

type PermissionVPEConfig struct {
	AllowedRole int
}

func PermissionAllow(role int) gin.HandlerFunc {
	return PermissionVPEWithConfig(PermissionVPEConfig{
		AllowedRole: role,
	})
}

func PermissionVPEWithConfig(config PermissionVPEConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountId, accountRole, err := ctxAccountInfo(c)
		if err != nil {
			mlog.Errorw("error", err.Error())
			response.GinJSONError(c, ErrorUnauthorized)
			return
		}

		if config.AllowedRole < accountRole {
			mlog.Errorw("accountId", accountId, "accountRole", accountRole, "error", errors.Forbidden("role %d is not allowed to access", accountRole))
			response.GinJSONError(c, ErrorForbidden)
			return
		}

		c.Next()
	}
}
