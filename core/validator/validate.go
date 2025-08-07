package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTrans "github.com/go-playground/validator/v10/translations/en"
	zhTrans "github.com/go-playground/validator/v10/translations/zh"

	"github.com/kochabonline/kit/log"
)

const (
	LanguageEN         = "en"
	LanguageZH         = "zh"
	ErrorSeparator     = "; "
	TranslationTag     = "label"
	TranslationJSONTag = "json"
)

var (
	ErrTranslatorNotFound = errors.New("translator not found")
)

var (
	defaultValidator *Validator
)

type Validator struct {
	validate *validator.Validate
	trans    map[string]ut.Translator
}

type Option func(*Validator)

// NewValidator 创建一个具有默认设置的新验证器实例
func NewValidator(opts ...Option) *Validator {
	validator := &Validator{
		validate: validator.New(),
		trans:    make(map[string]ut.Translator),
	}

	for _, opt := range opts {
		opt(validator)
	}

	validator.initialize()

	return validator
}

func (v *Validator) initialize() {
	// 创建英文和中文翻译器
	enTranslator := en.New()
	zhTranslator := zh.New()

	uni := ut.New(enTranslator, zhTranslator)

	// 注册英文和中文翻译器
	transEn, _ := uni.GetTranslator(LanguageEN)
	transZh, _ := uni.GetTranslator(LanguageZH)

	v.trans = map[string]ut.Translator{
		LanguageEN: transEn,
		LanguageZH: transZh,
	}

	// 注册验证器的翻译
	if err := enTrans.RegisterDefaultTranslations(v.validate, transEn); err != nil {
		log.Error().Err(err).Msg("failed to register English translations")
	}
	if err := zhTrans.RegisterDefaultTranslations(v.validate, transZh); err != nil {
		log.Error().Err(err).Msg("failed to register Chinese translations")
	}

	v.validate.RegisterTagNameFunc(v.getFieldName)
}

// getFieldName 从结构体标签中提取字段名称
func (v *Validator) getFieldName(fld reflect.StructField) string {
	// 首先尝试从 label 标签获取
	if label := fld.Tag.Get(TranslationTag); label != "" {
		return label
	}

	// 然后尝试从 json 标签获取
	if jsonTag := fld.Tag.Get(TranslationJSONTag); jsonTag != "" {
		name := strings.SplitN(jsonTag, ",", 2)[0]
		if name != "-" && name != "" {
			return name
		}
	}

	// 回退到字段名称
	return fld.Name
}

func (v *Validator) GetValidate() *validator.Validate {
	return v.validate
}

func (v *Validator) GetTranslator(language string) (ut.Translator, error) {
	if language == "" {
		language = LanguageEN
	}

	trans, found := v.trans[language]
	if !found {
		return nil, ErrTranslatorNotFound
	}
	return trans, nil
}

func (v *Validator) StructTrans(target any, language string) error {
	var trans ut.Translator
	var err error

	if trans, err = v.GetTranslator(language); err != nil {
		return err
	}

	err = v.validate.Struct(target)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return err
		}

		// 翻译验证错误信息
		sb := strings.Builder{}
		for _, e := range err.(validator.ValidationErrors) {
			if sb.Len() > 0 {
				sb.WriteString(ErrorSeparator)
			}
			sb.WriteString(e.Translate(trans))
		}
		return errors.New(sb.String())
	}
	return nil
}

func (v *Validator) Struct(target any) error {
	return v.StructTrans(target, LanguageZH)
}

func init() {
	defaultValidator = NewValidator()
}

func GetValidate() *validator.Validate {
	return defaultValidator.GetValidate()
}

func GetTranslator(language string) (ut.Translator, error) {
	return defaultValidator.GetTranslator(language)
}

func Struct(target any) error {
	return defaultValidator.Struct(target)
}

func StructTrans(target any, language string) error {
	return defaultValidator.StructTrans(target, language)
}
