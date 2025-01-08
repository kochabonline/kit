package gotp

import (
	"testing"
)

func Test(t *testing.T) {
	ga := NewGoogleAuthenticator()
	secret := ga.GenerateSecret()
	code, _ := ga.GenerateCode(secret)
	qr := ga.GenerateQRUrl("test", "test@gmail.com", secret)
	ok, _ := ga.ValidateCode(secret, code)

	t.Log(secret, code, qr, ok)
}
