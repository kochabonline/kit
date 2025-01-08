package ecies

import (
	"bytes"
	"crypto/elliptic"
	"crypto/subtle"
	"encoding/hex"
	"math/big"
)

type PublicKey struct {
	elliptic.Curve
	X, Y *big.Int
}

// Bytes returns public key raw bytes
// Could be optionally compressed by dropping Y part
func (pub *PublicKey) Bytes(compressed bool) []byte {
	x := pub.X.Bytes()
	x = zeroPad(x, 32)

	if compressed {
		if pub.Y.Bit(0) == 0 {
			return bytes.Join([][]byte{{0x03}, x}, nil)
		}
		return bytes.Join([][]byte{{0x02}, x}, nil)
	}

	y := pub.Y.Bytes()
	y = zeroPad(y, 32)

	return bytes.Join([][]byte{{0x04}, x, y}, nil)
}

// Hex returns public key bytes in hex form
func (pub *PublicKey) Hex(compressed bool) string {
	return hex.EncodeToString(pub.Bytes(compressed))
}

// Decapsulate decapsulates key by using Key Encapsulation Mechanism and returns symmetric key;
// can be safely used as encryption key
func (pub *PublicKey) Decapsulate(priv *PrivateKey) ([]byte, error) {
	if !pub.Curve.IsOnCurve(pub.X, pub.Y) {
		return nil, ErrInvalidPublicKey
	}

	if priv == nil {
		return nil, ErrPrivateKeyEmpty
	}

	var secret bytes.Buffer
	secret.Write(pub.Bytes(false))
	sx, sy := priv.Curve.ScalarMult(pub.X, pub.Y, priv.D.Bytes())
	secret.Write([]byte{0x04})

	l := len(priv.Curve.Params().P.Bytes())
	secret.Write(zeroPad(sx.Bytes(), l))
	secret.Write(zeroPad(sy.Bytes(), l))

	return kdf(secret.Bytes())
}

// Equals compares two public keys with constant time (to resist timing attacks)
func (pub *PublicKey) Equals(k *PublicKey) bool {
	eqX := subtle.ConstantTimeCompare(pub.X.Bytes(), k.X.Bytes()) == 1
	eqY := subtle.ConstantTimeCompare(pub.Y.Bytes(), k.Y.Bytes()) == 1
	return eqX && eqY
}
