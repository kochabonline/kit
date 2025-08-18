package validator

import (
	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

// 默认语言头名称
const DefaultLanguageHeader = "Language"

// 当前语言头名称
var languageHeader = DefaultLanguageHeader

// GetLanguageHeader 返回当前配置的语言头名称
func GetLanguageHeader() string {
	return languageHeader
}

// SetLanguageHeader 设置语言头名称
func SetLanguageHeader(header string) {
	if header != "" {
		languageHeader = header
	}
}

// bindValidate 是所有绑定函数的核心实现，提供统一的绑定和验证逻辑
func bindValidate(c *gin.Context, obj any, bindFunc func(any) error) error {
	// 执行数据绑定
	if err := bindFunc(obj); err != nil {
		return err
	}

	// 获取语言设置并进行验证
	language := c.GetHeader(GetLanguageHeader())
	if err := StructTrans(obj, language); err != nil {
		return errors.BadRequest("%s", err.Error())
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
