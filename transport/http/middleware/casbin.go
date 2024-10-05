package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

const (
	ErrCasbinForbidden = "access forbidden"
)

type CasbinConfig struct {
	E *casbin.SyncedCachedEnforcer
}

func Casbin(e *casbin.SyncedCachedEnforcer) gin.HandlerFunc {
	return CasbinWithConfig(CasbinConfig{
		E: e,
	})
}

func CasbinWithConfig(config CasbinConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.E == nil {
			c.Next()
			return
		}

		obj := c.Request.URL.Path
		act := c.Request.Method
		user, sub, err := userInfo(c)
		if err != nil {
			log.Errorw("user", user, "role", sub, "path", obj, "method", act, "error", err.Error())
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden, err.Error()))
			return
		}

		ok, err := config.E.Enforce(sub, obj, act)
		if err != nil {
			log.Errorw("user", user, "role", sub, "path", obj, "method", act, "error", err.Error())
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden, err.Error()))
			return
		}
		if !ok {
			log.Errorw("user", user, "role", sub, "path", obj, "method", act, "error", "casbin denied access")
			response.GinJSONError(c, errors.Forbidden(ErrCasbinForbidden, "casbin denied access"))
			return
		}

		c.Next()
	}
}
