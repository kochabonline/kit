package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

const (
	ErrCasbinForbidden = "forbidden"
)

type CasbinConfig struct {
	// SkippedPathPrefixes is a list of path prefixes that should be skipped from casbin
	SkippedPathPrefixes []string
	// E is the casbin synced cached enforcer
	E *casbin.SyncedCachedEnforcer
}

func CasbinWithConfig(config CasbinConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.E == nil || skippedPathPrefixes(c, config.SkippedPathPrefixes...) {
			c.Next()
			return
		}

		obj := c.Request.RequestURI
		act := c.Request.Method
		userId, sub, err := userInfo(c)
		if err != nil {
			log.Errorw("userId", userId, "role", sub, "path", obj, "method", act, "error", err.Error())
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden))
			return
		}

		ok, err := config.E.Enforce(sub, obj, act)
		if err != nil {
			log.Errorw("userId", userId, "role", sub, "path", obj, "method", act, "error", err.Error())
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden))
			return
		}
		if !ok {
			log.Errorw("userId", userId, "role", sub, "path", obj, "method", act, "error", "casbin denied access")
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden))
			return
		}

		c.Next()
	}
}
