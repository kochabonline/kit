package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/kochabonline/kit/errors"
)

type Option struct {
	data string
}

func WithData(data string) func(*Option) {
	return func(o *Option) {
		o.data = data
	}
}

func Sign(secret string, opts ...func(*Option)) (string, int64) {
	opt := &Option{}
	for _, o := range opts {
		o(opt)
	}

	timestamp := time.Now().Unix()
	var toSign strings.Builder
	toSign.WriteString(strconv.FormatInt(timestamp, 10))
	toSign.WriteString("\n")
	toSign.WriteString(secret)
	if opt.data != "" {
		toSign.WriteString(opt.data)
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(toSign.String()))
	return hex.EncodeToString(h.Sum(nil)), timestamp
}

func Verify(secret, signature string, timestamp int64, opts ...func(*Option)) error {
	opt := &Option{}
	for _, o := range opts {
		o(opt)
	}

	if time.Now().Unix()-timestamp > 60*5 {
		return errors.BadRequest("timestamp expired")
	}
	var toSign strings.Builder
	toSign.WriteString(strconv.FormatInt(timestamp, 10))
	toSign.WriteString("\n")
	toSign.WriteString(secret)
	if opt.data != "" {
		toSign.WriteString(opt.data)
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(toSign.String()))
	if hex.EncodeToString(h.Sum(nil)) != signature {
		return errors.BadRequest("signature mismatch")
	}

	return nil
}
