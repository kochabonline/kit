package hmac

import "testing"

func TestHmac(t *testing.T) {
	secret := "123456"
	message := "hello"
	signature, timestamp := Sign(secret, WithData(message))
	if err := Verify(secret, signature, timestamp, WithData(message)); err != nil {
		t.Fatal(err)
	}
}
