package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/errors"
)

type Response struct {
	Code   int    `json:"code"`
	Reason string `json:"reason"`
	Data   any    `json:"data"`
}

func NewResponse(code int, reason string, data any) *Response {
	return &Response{
		Code:   code,
		Reason: reason,
		Data:   data,
	}
}

func GinJSON(c *gin.Context, data any) {
	c.JSON(http.StatusOK, NewResponse(http.StatusOK, "", data))
}

func GinJSONError(c *gin.Context, err error) {
	defer c.Abort()

	e := errors.FromError(err)
	httpCode := int(e.Code)

	if http.StatusText(httpCode) == "" {
		httpCode = http.StatusInternalServerError
	}

	c.JSON(httpCode, NewResponse(int(e.Code), e.Reason, nil))
}
