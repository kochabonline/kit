package validator

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTrans "github.com/go-playground/validator/v10/translations/en"
	zhTrans "github.com/go-playground/validator/v10/translations/zh"

	"github.com/kochabonline/kit/log"
)

// 常量定义
const (
	DefaultTagName     = "label"
	DefaultJSONTagName = "json"
)

// 错误定义
var (
	ErrTranslatorNotFound = errors.New("translator not found")
	ErrInvalidLanguage    = errors.New("invalid language")
	ErrValidationFailed   = errors.New("validation failed")
)

// 全局实例
var (
	globalValidator Validator
	globalOnce      sync.Once
)

// validatorImpl 验证器实现
type validatorImpl struct {
	mu          sync.RWMutex
	validate    *validator.Validate
	translators map[Language]ut.Translator
	config      *Config
}

// NewValidator 创建新的验证器实例
func NewValidator(options ...Option) Validator {
	config := &Config{
		DefaultLanguage:    LanguageChinese,
		TagNameExtractor:   NewDefaultTagNameExtractor(),
		CustomValidations:  make(map[string]validator.Func),
		CustomTranslations: make(map[Language]map[string]TranslationPair),
	}

	// 应用配置选项
	for _, option := range options {
		option(config)
	}

	v := &validatorImpl{
		validate:    validator.New(),
		translators: make(map[Language]ut.Translator),
		config:      config,
	}

	v.initialize()
	return v
}

// initialize 初始化验证器
func (v *validatorImpl) initialize() {
	// 设置字段名提取器
	v.validate.RegisterTagNameFunc(v.extractFieldName)

	// 初始化翻译器
	v.initializeTranslators()

	// 注册自定义验证规则
	v.registerCustomValidations()

	// 注册自定义翻译
	v.registerCustomTranslations()
}

// initializeTranslators 初始化翻译器
func (v *validatorImpl) initializeTranslators() {
	enLocale := en.New()
	zhLocale := zh.New()
	// 使用英文作为fallback语言
	uni := ut.New(enLocale, enLocale, zhLocale)

	// 注册英文翻译器
	if transEn, found := uni.GetTranslator(LanguageEnglish.String()); found {
		v.translators[LanguageEnglish] = transEn
		if err := enTrans.RegisterDefaultTranslations(v.validate, transEn); err != nil {
			log.Error().Err(err).Msg("failed to register English translations")
		}
	} else {
		log.Error().Str("language", LanguageEnglish.String()).Msg("failed to get English translator")
	}

	// 注册中文翻译器
	if transZh, found := uni.GetTranslator(LanguageChinese.String()); found {
		v.translators[LanguageChinese] = transZh
		if err := zhTrans.RegisterDefaultTranslations(v.validate, transZh); err != nil {
			log.Error().Err(err).Msg("failed to register Chinese translations")
		}
	} else {
		log.Error().Str("language", LanguageChinese.String()).Msg("failed to get Chinese translator")
	}
}

// registerCustomValidations 注册自定义验证规则
func (v *validatorImpl) registerCustomValidations() {
	for tag, fn := range v.config.CustomValidations {
		if err := v.validate.RegisterValidation(tag, fn); err != nil {
			log.Error().Err(err).Str("tag", tag).Msg("failed to register custom validation")
		}
	}
}

// registerCustomTranslations 注册自定义翻译
func (v *validatorImpl) registerCustomTranslations() {
	for lang, translations := range v.config.CustomTranslations {
		translator, exists := v.translators[lang]
		if !exists {
			continue
		}

		for tag, pair := range translations {
			if err := v.validate.RegisterTranslation(tag, translator, pair.RegisterFunc, pair.TranslationFunc); err != nil {
				log.Error().Err(err).Str("tag", tag).Str("language", lang.String()).Msg("failed to register custom translation")
			}
		}
	}
}

// extractFieldName 提取字段名
func (v *validatorImpl) extractFieldName(fld reflect.StructField) string {
	// 首先尝试从 label 标签获取
	if label := fld.Tag.Get(DefaultTagName); label != "" {
		return label
	}

	// 然后尝试从 json 标签获取
	if jsonTag := fld.Tag.Get(DefaultJSONTagName); jsonTag != "" {
		name := strings.SplitN(jsonTag, ",", 2)[0]
		if name != "-" && name != "" {
			return name
		}
	}

	// 如果配置了自定义提取器，使用自定义提取器
	if v.config.TagNameExtractor != nil {
		// 这里需要创建一个临时的FieldError来适配接口
		// 实际使用中可能需要重新设计这个接口
		return fld.Name
	}

	// 回退到字段名称
	return fld.Name
}

// Validate 验证结构体
func (v *validatorImpl) Validate(ctx context.Context, data any) error {
	return v.ValidateWithLanguage(ctx, data, v.config.DefaultLanguage)
}

// ValidateWithLanguage 使用指定语言验证结构体
func (v *validatorImpl) ValidateWithLanguage(ctx context.Context, data any, lang Language) error {
	if !lang.IsValid() {
		return ErrInvalidLanguage
	}

	// 执行验证
	err := v.validate.Struct(data)
	if err == nil {
		return nil
	}

	// 处理验证错误
	var invalidValidationError *validator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return err
	}

	// 直接使用 validator.ValidationErrors
	// 调用者可以通过 GetTranslator 方法获取翻译器进行本地化处理
	if fieldErrors, ok := err.(validator.ValidationErrors); ok {
		return fieldErrors
	}

	return err
}

// RegisterValidation 注册自定义验证规则
func (v *validatorImpl) RegisterValidation(tag string, fn validator.Func) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.validate.RegisterValidation(tag, fn)
}

// RegisterTranslation 注册自定义翻译
func (v *validatorImpl) RegisterTranslation(tag string, lang Language, registerFn validator.RegisterTranslationsFunc, translationFn validator.TranslationFunc) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	translator, exists := v.translators[lang]
	if !exists {
		return ErrTranslatorNotFound
	}

	return v.validate.RegisterTranslation(tag, translator, registerFn, translationFn)
}

// GetValidator 获取底层的validator实例
func (v *validatorImpl) GetValidator() *validator.Validate {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.validate
}

// GetTranslator 获取指定语言的翻译器
func (v *validatorImpl) GetTranslator(lang Language) (ut.Translator, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	translator, exists := v.translators[lang]
	if !exists {
		return nil, ErrTranslatorNotFound
	}

	return translator, nil
}

// GetGlobalValidator 获取全局验证器实例
func GetGlobalValidator() Validator {
	globalOnce.Do(func() {
		globalValidator = NewValidator()
	})
	return globalValidator
}

// Validate 使用全局验证器验证结构体
func Validate(ctx context.Context, data any) error {
	return GetGlobalValidator().Validate(ctx, data)
}

// ValidateWithLanguage 使用全局验证器和指定语言验证结构体
func ValidateWithLanguage(ctx context.Context, data any, lang Language) error {
	return GetGlobalValidator().ValidateWithLanguage(ctx, data, lang)
}

// GetValidator 获取全局验证器的底层validator实例
func GetValidator() *validator.Validate {
	return GetGlobalValidator().GetValidator()
}

// GetTranslator 获取全局验证器的翻译器
func GetTranslator(lang Language) (ut.Translator, error) {
	return GetGlobalValidator().GetTranslator(lang)
}

// TranslateErrors 翻译验证错误
func TranslateErrors(errors validator.ValidationErrors, lang Language) map[string]string {
	translator, err := GetTranslator(lang)
	if err != nil {
		// 如果获取翻译器失败，返回原始错误消息
		result := make(map[string]string)
		for _, err := range errors {
			result[err.Field()] = err.Error()
		}
		return result
	}

	result := make(map[string]string)
	for _, err := range errors {
		result[err.Field()] = err.Translate(translator)
	}
	return result
}
