package gopt

import (
	"testing"
)

func Test(t *testing.T) {
	ga := NewGoogleAuthenticator()
	secret := ga.GenerateSecret()
	code, _ := ga.GenerateCode(secret)
	qr := ga.GenerateQRUrl("test", "test@gmail.com", secret)

	t.Log(secret, code, qr)
}