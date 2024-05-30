package ioc

import "github.com/gin-gonic/gin"

type Ioc interface {
	Init() error
	Name() string
}

type GinIRouter interface {
	Register(r gin.IRouter)
}
