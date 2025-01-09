package tools

import (
	"encoding/base64"
	"github.com/skip2/go-qrcode"
)

// QRCode generates a QR code and returns a Base64 encoded string
func QRCode(content string, size int) (string, error) {
	bytes, err := qrcode.Encode(content, qrcode.Medium, size)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
