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
