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
	// value must be string
	OperatorKey string
	// value must be slice of string
	OwnerKey string
}

func PermissionHPE(roles []string, operatorKey string, ownerKey string) gin.HandlerFunc {
	return PermissionHPEWithConfig(PermissionHPEConfig{
		SkippedRoles: roles,
		OperatorKey:  operatorKey,
		OwnerKey:     ownerKey,
	})
}

func PermissionHPEWithConfig(config PermissionHPEConfig) gin.HandlerFunc {
	if config.OperatorKey == "" {
		config.OperatorKey = "operator"
	}
	if config.OwnerKey == "" {
		config.OwnerKey = "owner"
	}

	return func(c *gin.Context) {
		_, accountRoles, err := ctxAccountInfo(c)
		if err != nil {
			log.Error().Err(err).Msg("get account info from context")
			response.GinJSONError(c, ErrorUnauthorized)
			return
		}

		if slice.ContainsSlice(config.SkippedRoles, accountRoles) {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		Operator, err := util.CtxValue[string](ctx, config.OperatorKey)
		if err != nil {
			log.Error().Err(err).Str("key", config.OperatorKey).Msg("operator key not found")
			response.GinJSONError(c, ErrorForbidden)
			return
		}
		Owner, err := util.CtxValue[[]string](ctx, config.OwnerKey)
		if err != nil {
			log.Error().Err(err).Str("key", config.OwnerKey).Msg("owner key not found")
			response.GinJSONError(c, ErrorForbidden)
			return
		}
		if !slice.Contains(Operator, Owner) {
			log.Error().Str("message", "operator is not owner").Str("operator", Operator).Strs("owner", Owner).Err(errors.Forbidden("operator %s is not allowed to access", Operator)).Msg("")
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
			log.Error().Err(err).Msg("get account info from context")
			response.GinJSONError(c, ErrorUnauthorized)
			return
		}

		if !slice.ContainsSlice(config.AllowedRoles, accountRoles) {
			log.Error().Int64("id", accountId).Strs("roles", accountRoles).Str("url", c.Request.URL.Path).Str("method", c.Request.Method).Err(errors.Forbidden("roles %s is not allowed to access", strings.Join(accountRoles, ","))).Msg("")
			response.GinJSONError(c, ErrorForbidden)
			return
		}

		c.Next()
	}
}
