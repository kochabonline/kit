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
	Param string
	// SkippedRoles is a list of roles that should be skipped from permission
	SkippedRoles []string
	// Validate is a function that takes a gin context and returns the auth value
	Validate func(c *gin.Context) (userId int64, userRole string, err error)
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

		var userId int64
		var userRole string
		var err error

		if config.Validate == nil {
			userId, userRole, err = userInfo(c)
			if err != nil {
				log.Errorw("error", err.Error())
				response.GinJSONError(c, err)
				return
			}
		} else {
			userId, userRole, err = config.Validate(c)
			if err != nil {
				log.Errorw("error", err.Error())
				response.GinJSONError(c, err)
				return
			}
		}

		if findRoleWithEmpty(userRole, config.SkippedRoles...) {
			c.Next()
			return
		}

		if paramValue != strconv.FormatInt(userId, 10) {
			log.Errorw("userId", userId, "error", errors.Forbidden(ErrPermissionForbidden, "%d is not allowed to access %s", userId, paramValue))
			response.GinJSONError(c, errors.Forbidden(ErrPermissionForbidden, "%d is not allowed to access %s", userId, paramValue))
			return
		}

		c.Next()
	}
}

type PermissionVPEConfig struct {
	// AllowRole is the role level that should be allowed to access
	AllowedRoles []string
	// Validate is a function that takes a gin context and returns the auth value
	Validate func(c *gin.Context) (userId int64, userRole string, err error)
}

func PermissionAllow(roles ...string) gin.HandlerFunc {
	return PermissionVPEWithConfig(PermissionVPEConfig{
		AllowedRoles: roles,
	})
}

func PermissionVPEWithConfig(config PermissionVPEConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userId int64
		var userRole string
		var err error

		if config.Validate == nil {
			userId, userRole, err = userInfo(c)
			if err != nil {
				log.Errorw("error", err.Error())
				response.GinJSONError(c, err)
				return
			}
		} else {
			userId, userRole, err = config.Validate(c)
			if err != nil {
				log.Errorw("error", err.Error())
				response.GinJSONError(c, err)
				return
			}
		}

		if !findRoleWithEmpty(userRole, config.AllowedRoles...) {
			log.Errorw("userId", userId, "userRole", userRole, "error", errors.Forbidden(ErrPermissionForbidden, "%s is not allowed to access", userRole))
			response.GinJSONError(c, errors.Forbidden(ErrPermissionForbidden, "%s is not allowed to access", userRole))
			return
		}

		c.Next()
	}
}
