package ecies

import (
	"bytes"
	"crypto/subtle"
	"encoding/hex"
	"math/big"
)

type PrivateKey struct {
	*PublicKey
	D *big.Int
}

func (priv *PrivateKey) Bytes() []byte {
	return priv.D.Bytes()
}

func (priv *PrivateKey) Hex() string {
	return hex.EncodeToString(priv.Bytes())
}

// Encapsulate encapsulates key by using Key Encapsulation Mechanism and returns symmetric key;
// can be safely used as encryption key
func (priv *PrivateKey) Encapsulate(pub *PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, ErrPublicKeyEmpty
	}

	if !pub.Curve.IsOnCurve(pub.X, pub.Y) {
		return nil, ErrInvalidPublicKey
	}

	var secret bytes.Buffer
	secret.Write(priv.PublicKey.Bytes(false))

	sx, sy := pub.Curve.ScalarMult(pub.X, pub.Y, priv.D.Bytes())
	secret.Write([]byte{0x04})

	l := len(pub.Curve.Params().P.Bytes())
	secret.Write(zeroPad(sx.Bytes(), l))
	secret.Write(zeroPad(sy.Bytes(), l))

	return kdf(secret.Bytes())
}

// ECDH derives shared secret;
// Must not be used as encryption key, it increases chances to perform successful key restoration attack
func (priv *PrivateKey) ECDH(pub *PublicKey) ([]byte, error) {
	if pub == nil {
		return nil, ErrPublicKeyEmpty
	}

	if !priv.Curve.IsOnCurve(pub.X, pub.Y) {
		return nil, ErrInvalidPublicKey
	}

	sx, sy := priv.Curve.ScalarMult(pub.X, pub.Y, priv.D.Bytes())

	var ss []byte
	if sy.Bit(0) != 0 { // if odd
		ss = append(ss, 0x03)
	} else { // if even
		ss = append(ss, 0x02)
	}

	l := len(pub.Curve.Params().P.Bytes())
	for i := 0; i < l-len(sx.Bytes()); i++ {
		ss = append(ss, 0x00)
	}

	return append(ss, sx.Bytes()...), nil
}

// Equals compares two private keys with constant time (to resist timing attacks)
func (priv *PrivateKey) Equals(k *PrivateKey) bool {
	return subtle.ConstantTimeCompare(priv.D.Bytes(), k.D.Bytes()) == 1
}
