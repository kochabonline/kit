package validator

import (
	"context"
	"testing"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// TestData 测试数据结构
type TestData struct {
	Name  string `json:"name" validate:"required,min=2" label:"姓名"`
	Age   int    `json:"age" validate:"required,min=1,max=120" label:"年龄"`
	Email string `json:"email" validate:"required,email" label:"邮箱"`
}

// AdultData 成人数据结构，用于测试自定义验证
type AdultData struct {
	Name string `json:"name" validate:"required,min=2" label:"姓名"`
	Age  int    `json:"age" validate:"required,adult_age"`
}

// 自定义验证函数：验证是否为成年人
func adultAgeValidator(fl validator.FieldLevel) bool {
	return fl.Field().Int() >= 18
}

// 中文翻译注册函数
var adultAgeRegisterZh = func(ut ut.Translator) error {
	return ut.Add("adult_age", "{0}必须是成年人(>=18岁)", true)
}

// 英文翻译注册函数
var adultAgeRegisterEn = func(ut ut.Translator) error {
	return ut.Add("adult_age", "{0} must be an adult (>=18 years old)", true)
}

// 翻译函数
var adultAgeTranslation = func(ut ut.Translator, fe validator.FieldError) string {
	t, _ := ut.T("adult_age", fe.Field())
	return t
}

func TestNewValidator(t *testing.T) {
	// 测试创建默认验证器
	v := NewValidator()
	if v == nil {
		t.Fatal("validator should not be nil")
	}

	// 测试验证器的基本功能
	data := TestData{
		Name:  "John Doe",
		Age:   25,
		Email: "john@example.com",
	}

	ctx := context.Background()
	err := v.Validate(ctx, data)
	if err != nil {
		t.Fatalf("validation should pass for valid data: %v", err)
	}
}

func TestValidatorWithCustomOptions(t *testing.T) {
	// 创建带有自定义选项的验证器
	v := NewValidator(
		WithDefaultLanguage(LanguageEnglish),
		WithCustomValidation("adult_age", adultAgeValidator),
		WithCustomTranslation("adult_age", LanguageChinese, adultAgeRegisterZh, adultAgeTranslation),
		WithCustomTranslation("adult_age", LanguageEnglish, adultAgeRegisterEn, adultAgeTranslation),
	)

	if v == nil {
		t.Fatal("validator should not be nil")
	}

	ctx := context.Background()

	// 测试有效数据
	validData := AdultData{
		Name: "Alice",
		Age:  25,
	}
	err := v.Validate(ctx, validData)
	if err != nil {
		t.Fatalf("validation should pass for valid adult data: %v", err)
	}

	// 测试无效数据（年龄小于18）
	invalidData := AdultData{
		Name: "Bob",
		Age:  16,
	}

	// 调试：记录测试信息
	t.Logf("Testing validation with age %d", invalidData.Age)

	err = v.ValidateWithLanguage(ctx, invalidData, LanguageChinese)
	if err == nil {
		t.Fatal("validation should fail for underage data")
	}

	// 检查错误类型
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		if len(validationErrors) != 1 {
			t.Fatalf("expected 1 validation error, got %d: %v", len(validationErrors), validationErrors)
		}

		if validationErrors[0].Tag() != "adult_age" {
			t.Fatalf("expected 'adult_age' tag, got '%s'", validationErrors[0].Tag())
		}

		// 获取翻译后的错误消息
		translator, _ := v.GetTranslator(LanguageChinese)
		t.Logf("Chinese error message: %s", validationErrors[0].Translate(translator))
	} else {
		t.Fatalf("expected validator.ValidationErrors type, got %T", err)
	}

	// 测试英文错误消息
	err = v.ValidateWithLanguage(ctx, invalidData, LanguageEnglish)
	if err == nil {
		t.Fatal("validation should fail for underage data")
	}

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		translator, _ := v.GetTranslator(LanguageEnglish)
		t.Logf("English error message: %s", validationErrors[0].Translate(translator))
	}
}

func TestValidationErrors(t *testing.T) {
	v := NewValidator()
	ctx := context.Background()

	// 测试多个验证错误
	invalidData := TestData{
		Name:  "A",             // 太短
		Age:   0,               // 小于最小值
		Email: "invalid-email", // 无效邮箱格式
	}

	err := v.Validate(ctx, invalidData)
	if err == nil {
		t.Fatal("validation should fail for invalid data")
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected validator.ValidationErrors type, got %T", err)
	}

	if len(validationErrors) == 0 {
		t.Fatal("expected validation errors")
	}

	// 测试错误集合的基本功能
	if len(validationErrors) == 0 {
		t.Fatal("should have validation errors")
	}

	// 测试获取特定字段的错误
	// 先打印所有错误的字段名以便调试
	for i, ve := range validationErrors {
		t.Logf("Error %d: Field=%s, Tag=%s", i, ve.Field(), ve.Tag())
	}

	// 查找Name字段的错误
	var nameErrors []validator.FieldError
	for _, ve := range validationErrors {
		if ve.Field() == "Name" || ve.Field() == "姓名" {
			nameErrors = append(nameErrors, ve)
		}
	}
	if len(nameErrors) == 0 {
		t.Fatal("expected errors for Name field")
	}

	// 测试错误消息
	errorMessage := validationErrors.Error()
	if errorMessage == "" {
		t.Fatal("error message should not be empty")
	}
	t.Logf("Validation errors: %s", errorMessage)

	// 测试翻译功能
	translatedErrors := TranslateErrors(validationErrors, LanguageChinese)
	t.Logf("Translated errors: %v", translatedErrors)
}

func TestLanguageType(t *testing.T) {
	// 测试Language类型的方法
	tests := []struct {
		lang  Language
		valid bool
		str   string
	}{
		{LanguageEnglish, true, "en"},
		{LanguageChinese, true, "zh"},
		{Language("fr"), false, "fr"},
		{Language(""), false, ""},
	}

	for _, test := range tests {
		if test.lang.IsValid() != test.valid {
			t.Errorf("expected IsValid() to return %v for %s", test.valid, test.lang)
		}
		if test.lang.String() != test.str {
			t.Errorf("expected String() to return %s for %s", test.str, test.lang)
		}
	}
}

func TestGlobalFunctions(t *testing.T) {
	ctx := context.Background()

	// 测试全局验证函数
	data := TestData{
		Name:  "Test User",
		Age:   30,
		Email: "test@example.com",
	}

	err := Validate(ctx, data)
	if err != nil {
		t.Fatalf("global Validate should pass for valid data: %v", err)
	}

	err = ValidateWithLanguage(ctx, data, LanguageEnglish)
	if err != nil {
		t.Fatalf("global ValidateWithLanguage should pass for valid data: %v", err)
	}

	// 测试获取全局验证器
	validator := GetValidator()
	if validator == nil {
		t.Fatal("GetValidator should not return nil")
	}

	// 测试获取翻译器
	translator, err := GetTranslator(LanguageChinese)
	if err != nil {
		t.Fatalf("GetTranslator should not return error: %v", err)
	}
	if translator == nil {
		t.Fatal("translator should not be nil")
	}
}

func TestConcurrentAccess(t *testing.T) {
	// 测试并发访问的安全性
	v := NewValidator()
	ctx := context.Background()

	data := TestData{
		Name:  "Concurrent Test",
		Age:   25,
		Email: "test@concurrent.com",
	}

	// 启动多个goroutine并发访问验证器
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				err := v.Validate(ctx, data)
				if err != nil {
					t.Errorf("concurrent validation failed: %v", err)
				}
			}
		}()
	}

	// 等待所有goroutine完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// 基准测试
func BenchmarkValidation(b *testing.B) {
	v := NewValidator()
	ctx := context.Background()

	data := TestData{
		Name:  "Benchmark Test",
		Age:   25,
		Email: "benchmark@test.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Validate(ctx, data)
	}
}

func BenchmarkValidationWithTranslation(b *testing.B) {
	v := NewValidator()
	ctx := context.Background()

	invalidData := TestData{
		Name:  "A", // 无效数据
		Age:   0,
		Email: "invalid",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateWithLanguage(ctx, invalidData, LanguageChinese)
	}
}
