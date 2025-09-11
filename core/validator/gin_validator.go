package validator

import (
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// BindFunc 定义绑定函数的类型
type BindFunc func(obj any) error

// GinConfig Gin验证器配置
type GinConfig struct {
	// LanguageHeader 用于获取语言偏好的HTTP头部字段名
	LanguageHeader string
	// DefaultLanguage 默认语言
	DefaultLanguage Language
	// Validator 验证器实例
	Validator Validator
}

// DefaultGinConfig 返回默认Gin配置
func DefaultGinConfig() *GinConfig {
	return &GinConfig{
		LanguageHeader:  "Accept-Language",
		DefaultLanguage: LanguageChinese,
		Validator:       GetGlobalValidator(),
	}
}

// GinValidator Gin验证器
type GinValidator struct {
	mu     sync.RWMutex
	config *GinConfig
}

// GinOption 定义Gin验证器选项函数类型
type GinOption func(*GinConfig)

// WithGinLanguageHeader 设置语言头部字段名选项
func WithGinLanguageHeader(header string) GinOption {
	return func(config *GinConfig) {
		config.LanguageHeader = header
	}
}

// WithGinDefaultLanguage 设置默认语言选项
func WithGinDefaultLanguage(language Language) GinOption {
	return func(config *GinConfig) {
		config.DefaultLanguage = language
	}
}

// WithGinValidator 设置验证器实例选项
func WithGinValidator(validator Validator) GinOption {
	return func(config *GinConfig) {
		config.Validator = validator
	}
}

// NewGinValidator 创建新的Gin验证器
func NewGinValidator(options ...GinOption) *GinValidator {
	config := DefaultGinConfig()

	// 应用所有选项
	for _, option := range options {
		option(config)
	}

	// 确保验证器实例不为空
	if config.Validator == nil {
		config.Validator = GetGlobalValidator()
	}

	return &GinValidator{
		config: config,
	}
}

// GetConfig 获取当前配置
func (gv *GinValidator) GetConfig() *GinConfig {
	gv.mu.RLock()
	defer gv.mu.RUnlock()

	// 返回配置的副本以避免并发修改
	return &GinConfig{
		LanguageHeader:  gv.config.LanguageHeader,
		DefaultLanguage: gv.config.DefaultLanguage,
		Validator:       gv.config.Validator,
	}
}

// UpdateConfig 使用选项函数更新配置
func (gv *GinValidator) UpdateConfig(options ...GinOption) {
	gv.mu.Lock()
	defer gv.mu.Unlock()

	// 应用所有选项到当前配置
	for _, option := range options {
		option(gv.config)
	}

	// 确保验证器实例不为空
	if gv.config.Validator == nil {
		gv.config.Validator = GetGlobalValidator()
	}
}

// bindAndValidate 是所有绑定函数的核心实现
func (gv *GinValidator) bindAndValidate(c *gin.Context, obj any, bindFunc BindFunc) error {
	config := gv.GetConfig()

	// 执行数据绑定
	if err := bindFunc(obj); err != nil {
		return err
	}

	// 获取语言设置
	language := gv.extractLanguage(c, config)

	// 进行结构体验证
	return config.Validator.ValidateWithLanguage(c.Request.Context(), obj, language)
}

// extractLanguage 提取语言设置
func (gv *GinValidator) extractLanguage(c *gin.Context, config *GinConfig) Language {
	// 从请求头获取语言
	langHeader := c.GetHeader(config.LanguageHeader)
	if langHeader != "" {
		// 处理语言变体，如 zh-CN -> zh, en-US -> en
		if len(langHeader) >= 2 {
			langCode := Language(langHeader[:2])
			if langCode.IsValid() {
				return langCode
			}
		}
	}

	// 回退到默认语言
	return config.DefaultLanguage
}

// ShouldBindHeader 绑定并验证 HTTP 头部数据
func (gv *GinValidator) ShouldBindHeader(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindHeader)
}

// ShouldBindUri 绑定并验证 URI 参数数据
func (gv *GinValidator) ShouldBindUri(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindUri)
}

// ShouldBindQuery 绑定并验证查询参数数据
func (gv *GinValidator) ShouldBindQuery(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindQuery)
}

// ShouldBindJSON 绑定并验证 JSON 数据
func (gv *GinValidator) ShouldBindJSON(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindJSON)
}

// ShouldBindXML 绑定并验证 XML 数据
func (gv *GinValidator) ShouldBindXML(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindXML)
}

// ShouldBindYAML 绑定并验证 YAML 数据
func (gv *GinValidator) ShouldBindYAML(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindYAML)
}

// ShouldBindTOML 绑定并验证 TOML 数据
func (gv *GinValidator) ShouldBindTOML(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBindTOML)
}

// ShouldBind 自动检测数据类型并绑定验证
func (gv *GinValidator) ShouldBind(c *gin.Context, obj any) error {
	return gv.bindAndValidate(c, obj, c.ShouldBind)
}

// ShouldBindWith 使用指定的绑定方式进行绑定验证
func (gv *GinValidator) ShouldBindWith(c *gin.Context, obj any, b binding.Binding) error {
	return gv.bindAndValidate(c, obj, func(obj any) error {
		return c.ShouldBindWith(obj, b)
	})
}

// 全局Gin验证器实例
var (
	globalGinValidator *GinValidator
	ginValidatorOnce   sync.Once
)

// GetGlobalGinValidator 获取全局Gin验证器实例
func GetGlobalGinValidator() *GinValidator {
	ginValidatorOnce.Do(func() {
		globalGinValidator = NewGinValidator()
	})
	return globalGinValidator
}

// 全局便捷函数

// ShouldBindHeader 使用全局Gin验证器绑定并验证 HTTP 头部数据
func ShouldBindHeader(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindHeader(c, obj)
}

// ShouldBindUri 使用全局Gin验证器绑定并验证 URI 参数数据
func ShouldBindUri(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindUri(c, obj)
}

// ShouldBindQuery 使用全局Gin验证器绑定并验证查询参数数据
func ShouldBindQuery(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindQuery(c, obj)
}

// ShouldBindJSON 使用全局Gin验证器绑定并验证 JSON 数据
func ShouldBindJSON(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindJSON(c, obj)
}

// ShouldBindXML 使用全局Gin验证器绑定并验证 XML 数据
func ShouldBindXML(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindXML(c, obj)
}

// ShouldBindYAML 使用全局Gin验证器绑定并验证 YAML 数据
func ShouldBindYAML(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindYAML(c, obj)
}

// ShouldBindTOML 使用全局Gin验证器绑定并验证 TOML 数据
func ShouldBindTOML(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBindTOML(c, obj)
}

// ShouldBind 使用全局Gin验证器自动检测数据类型并绑定验证
func ShouldBind(c *gin.Context, obj any) error {
	return GetGlobalGinValidator().ShouldBind(c, obj)
}

// ShouldBindWith 使用全局Gin验证器和指定的绑定方式进行绑定验证
func ShouldBindWith(c *gin.Context, obj any, b binding.Binding) error {
	return GetGlobalGinValidator().ShouldBindWith(c, obj, b)
}

// UpdateGinConfig 使用选项函数更新全局Gin验证器配置
func UpdateGinConfig(options ...GinOption) {
	GetGlobalGinValidator().UpdateConfig(options...)
}

// GetGinConfig 获取全局Gin验证器配置
func GetGinConfig() *GinConfig {
	return GetGlobalGinValidator().GetConfig()
}
