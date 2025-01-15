package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/core/util"
	"github.com/kochabonline/kit/core/util/slice"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/transport/http/response"
)

type PermissionHPEConfig struct {
	SkippedRoles []string
	OperatorKey  string
	OwnerKey     string
}

func PermissionHPE(roles []string, operatorKey string, ownerKey string) gin.HandlerFunc {
	return PermissionHPEWithConfig(PermissionHPEConfig{
		SkippedRoles: roles,
		OperatorKey:  operatorKey,
		OwnerKey:     ownerKey,
	})
}

func PermissionHPEWithConfig(config PermissionHPEConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, accountRoles, err := ctxAccountInfo(c)
		if err != nil {
			mlog.Errorw("error", err.Error())
			response.GinJSONError(c, ErrorUnauthorized)
			return
		}

		if slice.ContainsSlice(config.SkippedRoles, accountRoles) {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		OperatorValue, err := util.CtxValue[string](ctx, config.OperatorKey)
		if err != nil {
			mlog.Errorw("message", "operator key not found", "key", config.OperatorKey, "error", err.Error())
			response.GinJSONError(c, ErrorForbidden)
			return
		}
		OwnerKey, err := util.CtxValue[string](ctx, config.OwnerKey)
		if err != nil {
			mlog.Errorw("message", "owner key not found", "key", config.OwnerKey, "error", err.Error())
			response.GinJSONError(c, ErrorForbidden)
			return
		}
		if OperatorValue != OwnerKey {
			mlog.Errorw("operator", OperatorValue, "owner", OwnerKey, "error", errors.Forbidden("operator %s is not allowed to access", OperatorValue))
			response.GinJSONError(c, ErrorForbidden)
			return
		}

		c.Next()
	}
}

type PermissionVPEConfig struct {
	AllowedRoles []string
}

func PermissionAllow(roles []string) gin.HandlerFunc {
	return PermissionVPEWithConfig(PermissionVPEConfig{
		AllowedRoles: roles,
	})
}

func PermissionVPEWithConfig(config PermissionVPEConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountId, accountRoles, err := ctxAccountInfo(c)
		if err != nil {
			mlog.Errorw("error", err.Error())
			response.GinJSONError(c, ErrorUnauthorized)
			return
		}

		if !slice.ContainsSlice(config.AllowedRoles, accountRoles) {
			mlog.Errorw("accountId", accountId, "accountRoles", accountRoles, "error", errors.Forbidden("roles %s is not allowed to access", strings.Join(accountRoles, ",")))
			response.GinJSONError(c, ErrorForbidden)
			return
		}

		c.Next()
	}
}
