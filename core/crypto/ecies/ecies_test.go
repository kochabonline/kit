package ecies

import "testing"

func TestEcies(t *testing.T) {
	if err := GenerateKey(); err != nil {
		t.Fatal(err)
	}
	privateKey, err := LoadPrivateKey("private.pem")
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("hello, world!")
	ciphertext, err := Encrypt(&privateKey.PublicKey, msg)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ciphertext:", ciphertext)

	plaintext, err := Decrypt(privateKey, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("plaintext:", string(plaintext))
}
