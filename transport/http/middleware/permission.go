package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
	"github.com/kochabonline/kit/transport/http/response"
)

const (
	defaultParam = "id"
)

const (
	ErrPermissionForbidden = "forbidden"
)

type PermissionHPEConfig struct {
	// Param is the param key to look for the id value
	Param       string
	SkippedRole int
}

func PermissionHPE() gin.HandlerFunc {
	return PermissionHPEWithConfig(PermissionHPEConfig{})
}

func PermissionHPEWithConfig(config PermissionHPEConfig) gin.HandlerFunc {
	if config.Param == "" {
		config.Param = defaultParam
	}

	return func(c *gin.Context) {
		paramValue := c.Param(config.Param)
		if paramValue == "" {
			c.Next()
			return
		}

		userId, userRole, err := userInfo(c)
		if err != nil {
			log.Errorw("error", err.Error())
			response.GinJSONError(c, err)
			return
		}

		if config.SkippedRole <= userRole {
			c.Next()
			return
		}

		if paramValue != strconv.FormatInt(userId, 10) {
			log.Errorw("userId", userId, "error", errors.Forbidden("%d is not allowed to access %s", userId, paramValue))
			response.GinJSONError(c, errors.Forbidden(ErrPermissionForbidden))
			return
		}

		c.Next()
	}
}

type PermissionVPEConfig struct {
	AllowedRole int
}

func PermissionAllow(role int) gin.HandlerFunc {
	return PermissionVPEWithConfig(PermissionVPEConfig{
		AllowedRole: role,
	})
}

func PermissionVPEWithConfig(config PermissionVPEConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId, userRole, err := userInfo(c)
		if err != nil {
			log.Errorw("error", err.Error())
			response.GinJSONError(c, err)
			return
		}

		if config.AllowedRole >= userRole {
			log.Errorw("userId", userId, "userRole", userRole, "error", errors.Forbidden("%d is not allowed to access", userRole))
			response.GinJSONError(c, errors.Forbidden(ErrPermissionForbidden))
			return
		}

		c.Next()
	}
}
