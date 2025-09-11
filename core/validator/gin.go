package validator

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/kochabonline/kit/errors"
)

// GinBindOption Gin 绑定选项
type GinBindOption struct {
	LanguageHeader  string
	DefaultLanguage string
}

func DefaultGinBindOption() *GinBindOption {
	return &GinBindOption{
		LanguageHeader: "Accept-Language",
	}
}

// 配置读写锁，保护全局配置的并发访问
var (
	ginBindOptionMu sync.RWMutex
	ginBindOption   = DefaultGinBindOption()
)

// GetConfig 获取当前配置
func GetGinBindOption() *GinBindOption {
	ginBindOptionMu.RLock()
	defer ginBindOptionMu.RUnlock()
	return &GinBindOption{
		LanguageHeader: ginBindOption.LanguageHeader,
	}
}

// SetGinBindOption 设置配置
func SetGinBindOption(option *GinBindOption) {
	if option != nil {
		ginBindOptionMu.Lock()
		if option.LanguageHeader != "" {
			ginBindOption.LanguageHeader = option.LanguageHeader
		}
		if option.DefaultLanguage != "" {
			ginBindOption.DefaultLanguage = option.DefaultLanguage
		}
		ginBindOptionMu.Unlock()
	}
}

// BindFunc 定义绑定函数的类型
type BindFunc func(any) error

// bindValidate 是所有绑定函数的核心实现，提供统一的绑定和验证逻辑
// 它先执行数据绑定，然后根据请求头中的语言设置进行验证
func bindValidate(c *gin.Context, obj any, bindFunc BindFunc) error {
	opt := GetGinBindOption()

	// 执行数据绑定
	if err := bindFunc(obj); err != nil {
		return errors.BadRequest("failed to bind request data").WithCause(err)
	}

	// 获取语言设置，如果未设置则使用默认语言
	language := c.GetHeader(opt.LanguageHeader)
	if language == "" {
		language = opt.DefaultLanguage
	}

	// 进行结构体验证和错误翻译
	if err := StructTrans(obj, language); err != nil {
		return errors.BadRequest("failed structure validate").WithCause(err)
	}

	return nil
}

// GinShouldBindHeader 绑定并验证 HTTP 头部数据
func GinShouldBindHeader(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindHeader)
}

// GinShouldBindUri 绑定并验证 URI 参数数据
func GinShouldBindUri(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindUri)
}

// GinShouldBindQuery 绑定并验证查询参数数据
func GinShouldBindQuery(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindQuery)
}

// GinShouldBindJSON 绑定并验证 JSON 数据
func GinShouldBindJSON(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindJSON)
}

// GinShouldBind 自动检测数据类型并绑定验证
func GinShouldBind(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBind)
}

// GinShouldBindWith 使用指定的绑定方式进行绑定验证
func GinShouldBindWith(c *gin.Context, obj any, b binding.Binding) error {
	return bindValidate(c, obj, func(obj any) error {
		return c.ShouldBindWith(obj, b)
	})
}
