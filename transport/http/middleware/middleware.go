package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/core/util"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/log/zerolog"
)

var (
	ErrorForbidden    = errors.Forbidden("forbidden")
	ErrorUnauthorized = errors.Unauthorized("unauthorized")
)

var (
	mlog = log.NewHelper(zerolog.New())
)

func SetLogger(logger log.Logger) {
	mlog = log.NewHelper(logger)
}

func skippedPathPrefixes(c *gin.Context, prefixes ...string) bool {
	if len(prefixes) == 0 {
		return false
	}

	path := c.Request.URL.Path
	for _, prefix := range prefixes {
		if path == prefix || len(path) > len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

func ctxAccountInfo(c *gin.Context) (id int64, roles []string, err error) {
	ctx := c.Request.Context()
	id, err = util.CtxValue[int64](ctx, "id")
	if err != nil {
		return
	}
	roles, err = util.CtxValue[[]string](ctx, "roles")
	if err != nil {
		return
	}
	return
}
