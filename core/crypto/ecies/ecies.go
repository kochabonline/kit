package ecies

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"

	"github.com/kochabonline/kit/core/reflect"
)

var (
	ErrUnsupportedKeySize         = errors.New("ecies: unsupported key size, must be one of 224, 256, 384, 521")
	ErrInvalidDecode              = errors.New("ecies: invalid decode")
	ErrInvalidPublicKey           = errors.New("ecies: invalid public key")
	ErrSharedKeyIsPointAtInfinity = errors.New("ecies: shared key is point at infinity")
)

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
func LoadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
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

	return privateKey, nil
}

// LoadPublicKey loads an ECDSA public key from the specified file
func LoadPublicKey(path string) (*ecdsa.PublicKey, error) {
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

	return pub, nil
}

func Encrypt(pub *ecdsa.PublicKey, msg []byte) ([]byte, error) {
	tempPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tempPub := &tempPriv.PublicKey

	x, _ := pub.Curve.ScalarMult(pub.X, pub.Y, tempPriv.D.Bytes())
	if x == nil {
		return nil, ErrSharedKeyIsPointAtInfinity
	}
	sharedKey := sha256.Sum256(x.Bytes())

	ciphertext := make([]byte, len(msg))
	for i := range msg {
		ciphertext[i] = msg[i] ^ sharedKey[i%len(sharedKey)]
	}

	return append(append(tempPub.X.Bytes(), tempPub.Y.Bytes()...), ciphertext...), nil
}

func Decrypt(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	keyLen := (privateKey.Curve.Params().BitSize + 7) / 8
	tempPubX := new(big.Int).SetBytes(data[:keyLen])
	tempPubY := new(big.Int).SetBytes(data[keyLen : 2*keyLen])
	tempPub := &ecdsa.PublicKey{
		Curve: privateKey.Curve,
		X:     tempPubX,
		Y:     tempPubY,
	}

	x, _ := tempPub.Curve.ScalarMult(tempPub.X, tempPub.Y, privateKey.D.Bytes())
	if x == nil {
		return nil, ErrSharedKeyIsPointAtInfinity
	}
	sharedKey := sha256.Sum256(x.Bytes())

	msg := data[2*keyLen:]
	for i := range msg {
		msg[i] = msg[i] ^ sharedKey[i%len(sharedKey)]
	}

	return msg, nil
}
