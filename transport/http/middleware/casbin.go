package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

var (
	ErrCasbinForbiddenReason = "permission denied"
)

type CasbinConfig struct {
	E   *casbin.SyncedCachedEnforcer
	Sub func(c *gin.Context) (user string, role string, err error)
}

func CasbinWithConfig(config CasbinConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.E == nil || config.Sub == nil {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		method := c.Request.Method
		user, sub, err := config.Sub(c)
		if err != nil {
			handleError(c, user, errors.Forbidden(ErrCasbinForbiddenReason, "casbin failed to get subject: %v", err))
			return
		}
		ok, err := config.E.Enforce(sub, path, method)
		if err != nil {
			handleError(c, user, errors.Forbidden(ErrCasbinForbiddenReason, "casbin failed to enforce: %v", err))
			return
		}
		if !ok {
			handleError(c, user, errors.Forbidden(ErrCasbinForbiddenReason, "casbin denied access to %s %s for %s", method, path, sub))
			return
		}
		c.Next()
	}
}
