package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

const (
	ErrCasbinForbidden = "permission denied"
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
			log.Errorw("user", user, "error", err.Error())
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden, err.Error()))
			return
		}
		ok, err := config.E.Enforce(sub, path, method)
		if err != nil {
			log.Errorw("user", user, "error", err.Error())
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden, err.Error()))
			return
		}
		if !ok {
			log.Errorw("user", user, "error", "casbin denied access to %s %s for %s", method, path, sub)
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden, "casbin denied access to %s %s for %s", method, path, sub))
			return
		}

		c.Next()
	}
}
