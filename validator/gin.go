package validator

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var languageHeader atomic.Value

func init() {
	languageHeader.Store("Language")
}

func getLanguageHeader() string {
	return languageHeader.Load().(string)
}

func SetLanguageHeader(header string) {
	languageHeader.Store(header)
}

func bindValidate(c *gin.Context, obj any, bindFunc func(any) error) error {
	if err := bindFunc(obj); err != nil {
		return err
	}

	language := c.GetHeader(getLanguageHeader())

	return StructTrans(obj, language)
}

func GinShouldBindQuery(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindQuery)
}

func GinShouldBindJSON(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindJSON)
}

func GinShouldBind(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBind)
}
