package validator

import (
	"context"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// Language 定义支持的语言类型
type Language string

const (
	LanguageEnglish Language = "en"
	LanguageChinese Language = "zh"
)

// String 返回语言的字符串表示
func (l Language) String() string {
	return string(l)
}

// IsValid 检查语言是否有效
func (l Language) IsValid() bool {
	return l == LanguageEnglish || l == LanguageChinese
}

// Validator 定义验证器接口
type Validator interface {
	// Validate 验证结构体
	Validate(ctx context.Context, data any) error
	// ValidateWithLanguage 使用指定语言验证结构体
	ValidateWithLanguage(ctx context.Context, data any, lang Language) error
	// RegisterValidation 注册自定义验证规则
	RegisterValidation(tag string, fn validator.Func) error
	// RegisterTranslation 注册自定义翻译
	RegisterTranslation(tag string, lang Language, registerFn validator.RegisterTranslationsFunc, translationFn validator.TranslationFunc) error
	// GetValidator 获取底层的validator实例
	GetValidator() *validator.Validate
	// GetTranslator 获取指定语言的翻译器
	GetTranslator(lang Language) (ut.Translator, error)
}

// Config 验证器配置
type Config struct {
	// DefaultLanguage 默认语言
	DefaultLanguage Language
	// TagNameExtractor 字段名提取器
	TagNameExtractor TagNameExtractor
	// CustomValidations 自定义验证规则
	CustomValidations map[string]validator.Func
	// CustomTranslations 自定义翻译
	CustomTranslations map[Language]map[string]TranslationPair
}

// TranslationPair 翻译对
type TranslationPair struct {
	RegisterFunc    validator.RegisterTranslationsFunc
	TranslationFunc validator.TranslationFunc
}

// TagNameExtractor 定义字段名提取器接口
type TagNameExtractor interface {
	ExtractName(field validator.FieldError) string
}

// DefaultTagNameExtractor 默认字段名提取器
type DefaultTagNameExtractor struct{}

// NewDefaultTagNameExtractor 创建默认字段名提取器
func NewDefaultTagNameExtractor() TagNameExtractor {
	return &DefaultTagNameExtractor{}
}

// ExtractName 提取字段名
func (e *DefaultTagNameExtractor) ExtractName(field validator.FieldError) string {
	return field.Field()
}

// Option 配置选项函数
type Option func(*Config)

// WithDefaultLanguage 设置默认语言
func WithDefaultLanguage(lang Language) Option {
	return func(c *Config) {
		c.DefaultLanguage = lang
	}
}

// WithTagNameExtractor 设置字段名提取器
func WithTagNameExtractor(extractor TagNameExtractor) Option {
	return func(c *Config) {
		c.TagNameExtractor = extractor
	}
}

// WithCustomValidation 添加自定义验证规则
func WithCustomValidation(tag string, fn validator.Func) Option {
	return func(c *Config) {
		if c.CustomValidations == nil {
			c.CustomValidations = make(map[string]validator.Func)
		}
		c.CustomValidations[tag] = fn
	}
}

// WithCustomTranslation 添加自定义翻译
func WithCustomTranslation(tag string, lang Language, registerFn validator.RegisterTranslationsFunc, translationFn validator.TranslationFunc) Option {
	return func(c *Config) {
		if c.CustomTranslations == nil {
			c.CustomTranslations = make(map[Language]map[string]TranslationPair)
		}
		if c.CustomTranslations[lang] == nil {
			c.CustomTranslations[lang] = make(map[string]TranslationPair)
		}
		c.CustomTranslations[lang][tag] = TranslationPair{
			RegisterFunc:    registerFn,
			TranslationFunc: translationFn,
		}
	}
}
