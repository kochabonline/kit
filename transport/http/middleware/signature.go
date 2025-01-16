package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/transport/http/response"
)

type SignatureConfig struct {
	Secret       string
	EnableParams bool
	EnableBody   bool
	EnablePath   bool
	EnableMethod bool
}

func SignatureWithConfig(config SignatureConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the signature from the header
		header := c.GetHeader("Signature")
		if header == "" {
			mlog.Errorw("message", "signature", "error", "missing signature")
			response.GinJSONError(c, ErrorSignature)
			return
		}

		var toCompute strings.Builder

		// Get the parameters from the query string
		if config.EnableParams {
			var paramsString string
			params := c.Request.URL.Query()
			if len(params) > 0 {
				var keys []string
				for k := range params {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				var paramPairs []string
				for _, k := range keys {
					paramPairs = append(paramPairs, k+"="+params.Get(k))
				}
				paramsString = strings.Join(paramPairs, "&")
			}
			toCompute.WriteString(paramsString)
		}

		// Get the raw data from the request body
		if config.EnableBody {
			body, err := c.GetRawData()
			if err != nil {
				mlog.Errorw("message", "signature get raw data", "error", err.Error())
				response.GinJSONError(c, ErrorSignature)
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			toCompute.Write(body)
		}

		// Get the request path
		if config.EnablePath {
			toCompute.WriteString(c.Request.URL.Path)
		}

		// Get the request method
		if config.EnableMethod {
			toCompute.WriteString(c.Request.Method)
		}

		// Compute the signature
		signature := computeHMACSHA256(config.Secret, toCompute.String())
		if signature != header {
			mlog.Errorw("message", "signature", "error", "verify signature failed")
			response.GinJSONError(c, ErrorSignature)
			return
		}

		c.Next()
	}
}

func computeHMACSHA256(secret string, toComputeString string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(toComputeString))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
