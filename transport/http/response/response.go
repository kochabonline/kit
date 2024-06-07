package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

type Response struct {
	Code    int    `json:"code"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewResponse(code int, reason string, message string, data any) *Response {
	return &Response{
		Code:    code,
		Reason:  reason,
		Message: message,
		Data:    data,
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
	c.JSON(http.StatusOK, NewResponse(http.StatusOK, "", "", data))
}

func GinJSONError(c *gin.Context, err error) {
	defer c.Abort()

	e := errors.FromError(err)
	httpCode := int(e.Code)

	if http.StatusText(httpCode) == "" {
		httpCode = http.StatusInternalServerError
	}

	c.JSON(httpCode, NewResponse(int(e.Code), e.Reason, e.Message, nil))
}
