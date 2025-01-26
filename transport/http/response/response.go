package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

type Response struct {
	Code    int    `json:"code"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

func NewResponse(code int, data any, message string) *Response {
	return &Response{
		Code:    code,
		Data:    data,
		Message: message,
	}
}

func JSON(ctx any, data any) {
	if c, ok := ctx.(*gin.Context); ok {
		GinJSON(c, data)
	}
}

func JSONError(ctx any, err error) {
	if c, ok := ctx.(*gin.Context); ok {
		GinJSONError(c, err)
	}
}

func GinJSON(c *gin.Context, data any) {
	c.JSON(http.StatusOK, NewResponse(http.StatusOK, data, ""))
}

func GinJSONError(c *gin.Context, err error) {
	defer c.Abort()

	e := errors.FromError(err)
	httpCode := int(e.Code)

	// If http status text is empty, default to 200
	if http.StatusText(httpCode) == "" {
		httpCode = http.StatusInternalServerError
	}

	c.JSON(httpCode, NewResponse(int(e.Code), nil, e.Message))
}
