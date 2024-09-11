package middleware

import (
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

var (
	ErrCasbinUnauthorized = errors.Forbidden("permission denied", "casbin middleware failed to authorize request")
)

type CasbinConfig struct {
	E   *casbin.SyncedCachedEnforcer
	Sub func(c *gin.Context) (string, error)
}

func CasbinWithConfig(config CasbinConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method
		sub, err := config.Sub(c)
		if err != nil {
			log.Errorf("casbin failed to get subject: %v", err)
			response.GinJSONError(c, ErrCasbinUnauthorized)
			return
		}
		ok, err := config.E.Enforce(sub, path, method)
		if err != nil {
			log.Errorf("casbin failed to enforce: %v", err)
			response.GinJSONError(c, ErrCasbinUnauthorized)
			return
		}
		if !ok {
			log.Errorf("casbin denied access to %s %s for %s", method, path, sub)
			response.GinJSONError(c, ErrCasbinUnauthorized)
			return
		}
		c.Next()
	}
}
