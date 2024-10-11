package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type Jwt struct {
	config *Config
}

func New(config *Config) (*Jwt, error) {
	jwt := &Jwt{}

	if error := config.init(); error != nil {
		return nil, error
	}
	jwt.config = config

	return jwt, nil
}

func setClaim(claims map[string]any, key string, value any) {
	if value != nil && value != "" && value != 0 {
		claims[key] = value
	}
}

func (j *Jwt) generate(claims jwt.MapClaims, expire int64) (string, error) {
	now := time.Now().Unix()

	claims["exp"] = now + expire
	if j.config.IssuedAt != 0 {
		claims["iat"] = now + j.config.IssuedAt
	}
	if j.config.NotBefore != 0 {
		claims["nbf"] = now + j.config.NotBefore
	}
	setClaim(claims, "aud", j.config.Audience)
	setClaim(claims, "iss", j.config.Issuer)
	setClaim(claims, "sub", j.config.Subject)

	token := jwt.NewWithClaims(j.config.signingMethod(), claims)
	return token.SignedString([]byte(j.config.Secret))
}

func (j *Jwt) Generate(claims jwt.MapClaims) (string, error) {
	return j.generate(claims, j.config.Expire)
}

func (j *Jwt) GenerateRefreshToken(claims jwt.MapClaims) (string, error) {
	return j.generate(claims, j.config.RefreshExpire)
}

func (j *Jwt) Parse(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.config.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (j *Jwt) Refresh(tokenString string) (string, error) {
	claims, err := j.Parse(tokenString)
	if err != nil {
		return "", err
	}

	return j.Generate(claims)
}
