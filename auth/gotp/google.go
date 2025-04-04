package gotp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kochabonline/kit/core/util"
)

// otpauth://totp/{label}?secret={secret}&issuer={issuer}&algorithm={algorithm}&digits={digits}&period={period}
type GoogleAuthenticatorer interface {
	GenerateSecret() string
	GenerateCode(secret string) (string, error)
	GenerateQRCode(label string, issuer string, secret string) (string, error)
	ValidateCode(secret, code string) (bool, error)
}

var (
	GA GoogleAuthenticatorer
)

func init() {
	GA = NewGoogleAuthenticator()
}

type GoogleAuthenticator struct {
	SecretSize   int `json:"secret_size"`
	ExpireSecond int `json:"expire_second"`
	Digits       int `json:"digits"`
	QrCodeSize   int `json:"qr_code_size"`
}

type Option func(*GoogleAuthenticator)

func WithSecretSize(secretSize int) Option {
	return func(ga *GoogleAuthenticator) {
		ga.SecretSize = secretSize
	}
}

func WithExpireSecond(expireSecond int) Option {
	return func(ga *GoogleAuthenticator) {
		ga.ExpireSecond = expireSecond
	}
}

func WithDigits(digits int) Option {
	return func(ga *GoogleAuthenticator) {
		ga.Digits = digits
	}
}

func WithQrCodeSize(qrCodeSize int) Option {
	return func(ga *GoogleAuthenticator) {
		ga.QrCodeSize = qrCodeSize
	}
}

// NewGoogleAuthenticator creates a new GoogleAuthenticator
func NewGoogleAuthenticator(Opt ...Option) GoogleAuthenticatorer {
	ga := &GoogleAuthenticator{
		SecretSize:   16,
		ExpireSecond: 30,
		Digits:       6,
		QrCodeSize:   256,
	}

	for _, opt := range Opt {
		opt(ga)
	}

	return ga
}

// GenerateSecret generates a new secret
func (ga *GoogleAuthenticator) GenerateSecret() string {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.BigEndian, time.Now().UnixNano())
	return strings.ToUpper(ga.base32encode(ga.hmacSha1(buf.Bytes(), nil)))
}

// GenerateCode generates a new code
func (ga *GoogleAuthenticator) GenerateCode(secret string) (string, error) {
	// Decode the secret from base32
	secretBytes, err := ga.base32decode(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	// Generate a counter based on the current Unix timestamp
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], uint64(time.Now().Unix())/uint64(ga.ExpireSecond))

	// Generate a HMAC-SHA-1 hash using the secret as the key and the counter as the message
	hmacSha1 := hmac.New(sha1.New, secretBytes)
	hmacSha1.Write(counterBytes[:])
	hash := hmacSha1.Sum(nil)

	// Extract a 4-byte dynamic binary code from the hash
	offset := hash[len(hash)-1] & 0x0F
	dynamicBinaryCode := binary.BigEndian.Uint32(hash[offset : offset+4])

	// Convert the dynamic binary code to a dynamic verification code
	dynamicVerificationCode := dynamicBinaryCode % uint32(math.Pow10(ga.Digits))

	// Format the dynamic verification code as a string
	code := fmt.Sprintf(fmt.Sprintf("%%0%dd", ga.Digits), dynamicVerificationCode)

	return code, nil
}

// GenerateQRCode generates a new QR code
func (ga *GoogleAuthenticator) generateQRData(label string, issuer string, secret string) string {
	var builder strings.Builder
	builder.WriteString("otpauth://totp/")
	builder.WriteString(label)
	builder.WriteString("?secret=")
	builder.WriteString(secret)
	builder.WriteString("&issuer=")
	builder.WriteString(issuer)
	timestamp := time.Now().Unix()
	builder.WriteString("&timestamp=")
	builder.WriteString(strconv.FormatInt(timestamp, 10))

	return builder.String()
}

// GenerateQRrl generates a new QR code
func (ga *GoogleAuthenticator) GenerateQRCode(label string, issuer string, secret string) (string, error) {
	return util.QRCode(ga.generateQRData(label, issuer, secret), ga.QrCodeSize)
}

// ValidateCode validates a code
func (ga *GoogleAuthenticator) ValidateCode(secret, code string) (bool, error) {
	// Generate a code based on the current time and the secret
	generatedCode, err := ga.GenerateCode(secret)
	if err != nil {
		return false, err
	}

	// Compare the generated code with the provided code
	return generatedCode == code, nil
}

func (ga *GoogleAuthenticator) base32encode(bt []byte) string {
	return base32.StdEncoding.EncodeToString(bt)
}

func (ga *GoogleAuthenticator) base32decode(str string) ([]byte, error) {
	return base32.StdEncoding.DecodeString(str)
}

func (ga *GoogleAuthenticator) hmacSha1(key, bt []byte) []byte {
	h := hmac.New(sha1.New, key)
	if total := len(bt); total > 0 {
		h.Write(bt)
	}
	return h.Sum(nil)
}
