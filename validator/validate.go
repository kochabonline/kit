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

var (
	Validate *validator.Validate
	TransEn  ut.Translator
	TransZh  ut.Translator
)

func init() {
	initValidator()
}

// initValidator initializes the validator and translator.
func initValidator() {
	Validate = validator.New()

	// Create the English and Chinese translators.
	enTranslator := en.New()
	zhTranslator := zh.New()

	uni := ut.New(enTranslator, zhTranslator)

	// Register the English and Chinese translators.
	TransEn, _ = uni.GetTranslator("en")
	TransZh, _ = uni.GetTranslator("zh")

	// Register the default translations for the English translator.
	if err := enTrans.RegisterDefaultTranslations(Validate, TransEn); err != nil {
		log.Errorf("authenticator registration translator error: %v", err)
	}

	// Register the default translations for the Chinese translator.
	if err := zhTrans.RegisterDefaultTranslations(Validate, TransZh); err != nil {
		log.Errorf("authenticator registration translator error: %v", err)
	}

	// Register the tag name function.
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			name = ""
		}
		label := fld.Tag.Get("label")
		if label != "" {
			name = label
		}

		return name
	})
}

func RegisterValidation(tag string, fn validator.Func) error {
	return Validate.RegisterValidation(tag, fn)
}

func RegisterTranslation(tag string, trans ut.Translator, registerFn validator.RegisterTranslationsFunc, translationFn validator.TranslationFunc) error {
	return Validate.RegisterTranslation(tag, trans, registerFn, translationFn)
}

// Struct validates the given struct using the validator and the default translator.
func Struct(target any) error {
	return StructTrans(target, "zh")
}

// StructTrans validates the given struct using the validator and translator.
func StructTrans(target any, language string) error {
	err := Validate.Struct(target)
	if err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return err
		}

		var trans ut.Translator
		if strings.HasPrefix(language, "zh") {
			trans = TransZh
		} else if strings.HasPrefix(language, "en") {
			trans = TransEn
		} else {
			trans = TransEn
		}

		// Translate the validation errors.
		sb := strings.Builder{}
		for _, e := range err.(validator.ValidationErrors) {
			if sb.Len() > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(e.Translate(trans))
		}
		return errors.New(sb.String())
	}
	return nil
}
