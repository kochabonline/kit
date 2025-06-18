package middleware

import (
	"bytes"
	"encoding/base64"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/core/crypto/ecies"
	"github.com/kochabonline/kit/transport/http/response"
)

type CryptoConfig struct {
	PrivateKeyPath string
}

func CryptoWithConfig(config CryptoConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the request body
		body, err := c.GetRawData()
		if err != nil {
			mlog.Error().Err(err).Msg("read request body")
			response.GinJSONError(c, ErrorCrypto)
			return
		}
		// Decrypt the request body
		privateKey, err := ecies.LoadPrivateKey(config.PrivateKeyPath)
		if err != nil {
			mlog.Error().Err(err).Msg("crypto load private key")
			response.GinJSONError(c, ErrorCrypto)
			return
		}
		// Decode the base64 encoded ciphertext
		decodedCiphertext, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			mlog.Error().Err(err).Msg("crypto decode ciphertext")
			response.GinJSONError(c, ErrorCrypto)
			return
		}
		// Decrypt the ciphertext
		plaintext, err := ecies.Decrypt(privateKey, decodedCiphertext)
		if err != nil {
			mlog.Error().Err(err).Msg("crypto decrypt")
			response.GinJSONError(c, ErrorCrypto)
			return
		}
		// Write the plaintext back to the request body
		c.Request.Body = io.NopCloser(bytes.NewBuffer(plaintext))

		c.Next()
	}
}
