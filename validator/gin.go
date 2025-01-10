package validator

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
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
	if err := StructTrans(obj, language); err != nil {
		return errors.BadRequest("%v", err)
	}

	return nil
}

func GinShouldBindHeader(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindHeader)
}

func GinShouldBindUri(c *gin.Context, obj any) error {
	return bindValidate(c, obj, c.ShouldBindUri)
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
