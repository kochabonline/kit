package ecies

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"

	"os"
	"path/filepath"

	"github.com/kochabonline/kit/core/reflect"
	"golang.org/x/crypto/hkdf"
)

var (
	ErrInvalidDecode     = errors.New("ecies: invalid decode")
	ErrInvalidPublicKey  = errors.New("ecies: invalid public key")
	ErrPrivateKeyEmpty   = errors.New("ecies: private key is empty")
	ErrPublicKeyEmpty    = errors.New("ecies: public key is empty")
	ErrInvalidDataLength = errors.New("ecies: invalid data length")
)

func kdf(secret []byte) (key []byte, err error) {
	key = make([]byte, 32)
	kdf := hkdf.New(sha256.New, secret, nil, nil)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, fmt.Errorf("ecies: failed to derive key: %w", err)
	}

	return key, nil
}

func zeroPad(b []byte, length int) []byte {
	if len(b) > length {
		return b[:length]
	}
	if len(b) < length {
		b = append(make([]byte, length-len(b)), b...)
	}
	return b
}

// PublicKey represents an ECDSA public key
func ImportECDSAPublic(publicKey *ecdsa.PublicKey) *PublicKey {
	return &PublicKey{
		Curve: publicKey.Curve,
		X:     publicKey.X,
		Y:     publicKey.Y,
	}
}

// PrivateKey represents an ECDSA private key
func ImportECDSA(privateKey *ecdsa.PrivateKey) *PrivateKey {
	pub := ImportECDSAPublic(&privateKey.PublicKey)
	return &PrivateKey{PublicKey: pub, D: privateKey.D}
}

// GenerateKey Generates a new ECDSA key pair and saves it to the specified directory
func GenerateKey(opts ...func(*KeyOption)) error {
	option := &KeyOption{}
	// Set default tags
	if err := reflect.SetDefaultTag(option); err != nil {
		return err
	}
	// Apply options
	for _, opt := range opts {
		opt(option)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return err
	}
	privateKeyFile, err := os.Create(filepath.Join(option.Dirpath, option.PrivateKeyFilename))
	if err != nil {
		return err
	}
	defer privateKeyFile.Close()
	if err := pem.Encode(privateKeyFile, &pem.Block{
		Type:  "ECDSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}); err != nil {
		return err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	publicKeyFile, err := os.Create(filepath.Join(option.Dirpath, option.PublicKeyFilename))
	if err != nil {
		return err
	}
	defer publicKeyFile.Close()
	if err := pem.Encode(publicKeyFile, &pem.Block{
		Type:  "ECDSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	}); err != nil {
		return err
	}

	return nil
}

// LoadPrivateKey loads an ECDSA private key from the specified file
func LoadPrivateKey(path string) (*PrivateKey, error) {
	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(privateKeyBytes)
	if block == nil {
		return nil, ErrInvalidDecode
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return ImportECDSA(privateKey), nil
}

// LoadPublicKey loads an ECDSA public key from the specified file
func LoadPublicKey(path string) (*PublicKey, error) {
	publicKeyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(publicKeyBytes)
	if block == nil {
		return nil, ErrInvalidDecode
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pub, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, ErrInvalidPublicKey
	}

	return ImportECDSAPublic(pub), nil
}

func Encrypt(pub *PublicKey, data []byte) ([]byte, error) {
	var ct bytes.Buffer

	// ephemeral key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	ek := ImportECDSA(privateKey)

	ct.Write(ek.PublicKey.Bytes(false))

	// derive shared secret
	ss, err := ek.Encapsulate(pub)
	if err != nil {
		return nil, err
	}

	// aes encryption
	block, err := aes.NewCipher(ss)
	if err != nil {
		return nil, fmt.Errorf("ecies: cannot create aes block: %w", err)
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("ecies: cannot read random bytes for nonce: %w", err)
	}
	ct.Write(nonce)

	aesgcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return nil, fmt.Errorf("ecies: cannot create aes gcm: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, data, nil)

	tag := ciphertext[len(ciphertext)-aesgcm.NonceSize():]
	ct.Write(tag)
	ciphertext = ciphertext[:len(ciphertext)-len(tag)]
	ct.Write(ciphertext)

	return ct.Bytes(), nil
}

func Decrypt(priv *PrivateKey, data []byte) ([]byte, error) {
	if len(data) <= (1 + 32 + 32 + 16 + 16) {
		return nil, ErrInvalidDataLength
	}

	// ephemeral sender public key
	ethPub := &PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(data[1 : 1+32]),
		Y:     new(big.Int).SetBytes(data[1+32 : 1+32+32]),
	}

	// shift data
	data = data[65:]

	// derive shared secret
	ss, err := ethPub.Decapsulate(priv)
	if err != nil {
		return nil, err
	}

	// aes decryption part
	nonce := data[:16]
	tag := data[16:32]

	// ciphertext
	ciphertext := bytes.Join([][]byte{data[32:], tag}, nil)

	block, err := aes.NewCipher(ss)
	if err != nil {
		return nil, fmt.Errorf("ecies: cannot create new aes block: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return nil, fmt.Errorf("ecies: cannot create gcm cipher: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("ecies: cannot decrypt ciphertext: %w", err)
	}

	return plaintext, nil
}
