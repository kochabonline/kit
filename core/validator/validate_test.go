package validator

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type mock struct {
	Name string `json:"name" validate:"required,min=5" label:"姓名"`
	Age  int    `json:"age" validate:"required,adultAge" label:"年龄"`
}

func init() {
	transEn, err := GetTranslator(LanguageEN)
	if err != nil {
		panic(err)
	}
	transZh, err := GetTranslator(LanguageZH)
	if err != nil {
		panic(err)
	}

	GetValidate().RegisterValidation("adultAge", adultAge)
	GetValidate().RegisterTranslation("adultAge", transZh, registrationZhFunc, translateFunc)
	GetValidate().RegisterTranslation("adultAge", transEn, registrationEnFunc, translateFunc)
}

func adultAge(fl validator.FieldLevel) bool {
	return fl.Field().Int() >= 18
}

var registrationZhFunc = func(ut ut.Translator) error {
	return ut.Add("adultAge", "{0}必须是成年人", true)
}

var registrationEnFunc = func(ut ut.Translator) error {
	return ut.Add("adultAge", "{0} must be an adult", true)
}

var translateFunc = func(ut ut.Translator, fe validator.FieldError) string {
	t, _ := ut.T("adultAge", fe.Field())
	return t
}

func TestStruct(t *testing.T) {
	m := mock{
		Name: "test user", // 满足 min=5 的要求
		Age:  17,          // 满足 adultAge 的要求（>=18）
	}
	err := Struct(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m)
}

func TestBind(t *testing.T) {
	var m mock

	// mock request with valid data
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"name":"hello world!","age":15}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "zh")

	// mock context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// bind
	if err = GinShouldBindJSON(c, &m); err != nil {
		t.Fatal(err)
	}
	t.Log(m)
}
