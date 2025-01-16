package ecies

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEcies(t *testing.T) {
	// if err := GenerateKey(); err != nil {
	// 	t.Fatal(err)
	// }
	privateKey, err := LoadPrivateKey("private.pem")
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("hello, world!")
	msg = bytes.Repeat(msg, 10)
	ciphertext, err := Encrypt(privateKey.PublicKey, msg)
	if err != nil {
		t.Fatal(err)
	}
	encodedCiphertext := base64.StdEncoding.EncodeToString(ciphertext)
	t.Log("ciphertext:", encodedCiphertext)

	decodedCiphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := Decrypt(privateKey, decodedCiphertext)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("plaintext:", string(plaintext))
}
