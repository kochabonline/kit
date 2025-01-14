package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/transport/http/response"
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
		accountId, sub, err := ctxAccountInfo(c)
		if err != nil {
			mlog.Errorw("accountId", accountId, "role", sub, "path", obj, "method", act, "error", err.Error())
			response.GinJSONError(c, ErrorForbidden)
			return
		}

		ok, err := config.E.Enforce(sub, obj, act)
		if err != nil {
			mlog.Errorw("accountId", accountId, "role", sub, "path", obj, "method", act, "error", err.Error())
			response.GinJSONError(c, ErrorForbidden)
			return
		}
		if !ok {
			mlog.Errorw("accountId", accountId, "role", sub, "path", obj, "method", act, "error", "casbin denied access")
			response.GinJSONError(c, ErrorForbidden)
			return
		}

		c.Next()
	}
}
