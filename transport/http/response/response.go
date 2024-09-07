package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

type Response struct {
	Code    int    `json:"code"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
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

	// If http status text is empty, default to 500
	if http.StatusText(httpCode) == "" {
		httpCode = http.StatusInternalServerError
	}

	// Only return message for client errors
	var msg string
	if httpCode >= 400 && httpCode < 500 {
		msg = e.Message
	}

	c.JSON(httpCode, NewResponse(int(e.Code), e.Reason, msg, nil))
}
