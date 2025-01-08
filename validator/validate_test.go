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
	Validate.RegisterValidation("adultAge", adultAge)
	Validate.RegisterTranslation("adultAge", TransZh, registrationZhFunc, translateFunc)
	Validate.RegisterTranslation("adultAge", TransEn, registrationEnFunc, translateFunc)
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
		Name: "test",
		Age:  17,
	}
	err := Struct(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m)
}

func TestBind(t *testing.T) {
	var m mock

	// mock request
	req, err := http.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`{"name":"hello!","age":17}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Language", "en")

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
