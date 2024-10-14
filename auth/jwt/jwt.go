package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Jwt struct {
	config *Config
}

func New(config *Config) (*Jwt, error) {
	if err := config.init(); err != nil {
		return nil, err
	}
	return &Jwt{config: config}, nil
}

type Option struct {
	jti string
}

func WithJti(jti string) func(*Option) {
	return func(o *Option) {
		o.jti = jti
	}
}

func GetJti(claims jwt.MapClaims) (string, error) {
	jti, ok := claims["jti"].(string)
	if !ok {
		return "", jwt.ErrTokenInvalidId
	}
	return jti, nil
}

func setClaim(claims map[string]any, key string, value any) {
	if value != nil && value != "" && value != 0 {
		claims[key] = value
	}
}

func (j *Jwt) generate(claims jwt.MapClaims, expire int64, opts ...func(*Option)) (string, error) {
	option := &Option{}
	for _, opt := range opts {
		opt(option)
	}

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
	setClaim(claims, "jti", option.jti)

	token := jwt.NewWithClaims(j.config.signingMethod(), claims)
	return token.SignedString([]byte(j.config.Secret))
}

func (j *Jwt) Generate(claims jwt.MapClaims, opts ...func(*Option)) (string, error) {
	return j.generate(claims, j.config.Expire, opts...)
}

func (j *Jwt) GenerateRefreshToken(claims jwt.MapClaims, opts ...func(*Option)) (string, error) {
	return j.generate(claims, j.config.RefreshExpire, opts...)
}

// jwt.MapClaims is a type alias for map[string]interface{}
// Numeric values are converted to float64 during JSON unmarshalling
func (j *Jwt) Parse(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.config.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenNotValidYet
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

func (j *Jwt) Refresh(tokenString string, opts ...func(*Option)) (string, error) {
	claims, err := j.Parse(tokenString)
	if err != nil {
		return "", err
	}

	return j.Generate(claims, opts...)
}
