package jwt

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Auth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type JwtRedis struct {
	jwt           *Jwt
	client        *redis.Client
	idkey         string
	prefix        string
	accessPrefix  string
	refreshPrefix string
	multipleLogin bool
	expire        time.Duration
	refreshExpire time.Duration
}

type JwtRedisOption func(*JwtRedis)

func WithIdKey(idkey string) JwtRedisOption {
	return func(j *JwtRedis) {
		j.idkey = idkey
	}
}

// WithPrefix set prefix for redis key
func WithPrefix(prefix string) JwtRedisOption {
	return func(j *JwtRedis) {
		j.prefix = prefix
	}
}

// WithAccessPrefix set access prefix for redis key
func WithAccessPrefix(accessPrefix string) JwtRedisOption {
	return func(j *JwtRedis) {
		j.accessPrefix = accessPrefix
	}
}

// WithRefreshPrefix set refresh prefix for redis key
func WithRefreshPrefix(refreshPrefix string) JwtRedisOption {
	return func(j *JwtRedis) {
		j.refreshPrefix = refreshPrefix
	}
}

// WithMultipleLogin set multiple login
func WithMultipleLogin(multipleLogin bool) JwtRedisOption {
	return func(j *JwtRedis) {
		j.multipleLogin = multipleLogin
	}
}

func NewJwtRedis(jwt *Jwt, client *redis.Client, opts ...JwtRedisOption) *JwtRedis {
	expire := time.Duration(jwt.config.Expire) * time.Second
	refreshExpire := time.Duration(jwt.config.RefreshExpire) * time.Second

	jwtRedis := &JwtRedis{
		jwt:           jwt,
		client:        client,
		idkey:         "id",
		prefix:        "jwt",
		accessPrefix:  "access",
		refreshPrefix: "refresh",
		multipleLogin: false,
		expire:        expire,
		refreshExpire: refreshExpire,
	}

	for _, opt := range opts {
		opt(jwtRedis)
	}

	return jwtRedis
}

func (r *JwtRedis) key(id int64, key string) string {
	var builder strings.Builder
	builder.WriteString(r.prefix)
	builder.WriteString(":")
	builder.WriteString(strconv.FormatInt(id, 10))
	builder.WriteString(":")
	builder.WriteString(key)
	return builder.String()
}

func (r *JwtRedis) getIdByClaims(claims jwt.MapClaims) (int64, error) {
	id := JwtMapClaimsParse[int64](claims, r.idkey)
	if id == 0 {
		return 0, fmt.Errorf("invalid id, expected %s", r.idkey)
	}
	return id, nil
}

// exists checks if jti exists in redis, used to determine if the token is valid
func (r *JwtRedis) exists(ctx context.Context, id int64, jtis ...string) bool {
	if r.multipleLogin {
		keys := make([]string, len(jtis))
		for i, jti := range jtis {
			keys[i] = r.key(id, jti)
		}

		result, err := r.client.Exists(ctx, keys...).Result()
		if err != nil {
			return false
		}

		return result == int64(len(jtis))
	} else {
		keys := []string{
			r.key(id, r.accessPrefix),
			r.key(id, r.refreshPrefix),
		}
		result, err := r.client.MGet(ctx, keys...).Result()
		if err != nil {
			return false
		}

		resultMap := make(map[string]struct{}, len(result))
		for _, cmd := range result {
			if cmd != nil {
				resultMap[cmd.(string)] = struct{}{}
			}
		}

		for _, jti := range jtis {
			if _, exists := resultMap[jti]; exists {
				return true
			}
		}
	}

	return false
}

func (r *JwtRedis) Generate(ctx context.Context, claims jwt.MapClaims) (*Auth, error) {
	id, err := r.getIdByClaims(claims)
	if err != nil {
		return nil, err
	}

	aJti := uuid.New().String()
	rJti := uuid.New().String()

	aTokenString, err := r.jwt.Generate(claims, WithJti(aJti))
	if err != nil {
		return nil, err
	}
	rTokenString, err := r.jwt.GenerateRefreshToken(claims, WithJti(rJti))
	if err != nil {
		return nil, err
	}

	_, err = r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		if r.multipleLogin {
			pipe.Set(ctx, r.key(id, aJti), "", r.expire)
			pipe.Set(ctx, r.key(id, rJti), "", r.refreshExpire)
		} else {
			pipe.Set(ctx, r.key(id, r.accessPrefix), aJti, r.expire)
			pipe.Set(ctx, r.key(id, r.refreshPrefix), rJti, r.refreshExpire)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Auth{
		AccessToken:  aTokenString,
		RefreshToken: rTokenString,
	}, nil
}

func (r *JwtRedis) Refresh(ctx context.Context, token string) (string, error) {
	claims, err := r.jwt.Parse(token)
	if err != nil {
		return "", err
	}

	id, err := r.getIdByClaims(claims)
	if err != nil {
		return "", err
	}

	jti, err := GetJti(claims)
	if err != nil {
		return "", err
	}

	// jti does not exist in redis, indicating that the refresh token has expired
	if exists := r.exists(ctx, id, jti); !exists {
		return "", jwt.ErrTokenExpired
	}

	newJti := uuid.New().String()
	newToken, err := r.jwt.Generate(claims, WithJti(newJti))
	if err != nil {
		return "", err
	}

	if r.multipleLogin {
		return newToken, r.client.Set(ctx, r.key(id, newJti), "", r.expire).Err()
	}

	return newToken, r.client.Set(ctx, r.key(id, r.accessPrefix), newJti, r.expire).Err()
}

func (r *JwtRedis) Parse(ctx context.Context, tokens ...string) (jwt.MapClaims, error) {
	var err error
	claimsSlice := make([]jwt.MapClaims, len(tokens))
	for i, token := range tokens {
		claimsSlice[i], err = r.jwt.Parse(token)
		if err != nil {
			return nil, err
		}
	}

	jtis := make([]string, len(claimsSlice))
	for i, claims := range claimsSlice {
		jti, err := GetJti(claims)
		if err != nil {
			return nil, err
		}
		jtis[i] = jti
	}

	// The fields in the token are the same except for jti and exp, so you can take the first claims
	claims := claimsSlice[0]

	id, err := r.getIdByClaims(claimsSlice[0])
	if err != nil {
		return nil, err
	}

	if exists := r.exists(ctx, id, jtis...); !exists {
		return nil, jwt.ErrTokenExpired
	}

	return claims, nil
}

func (r *JwtRedis) Delete(ctx context.Context, id int64) error {
	var cursor uint64
	var keys []string
	var err error
	for {
		var scanKeys []string
		scanKeys, cursor, err = r.client.Scan(ctx, cursor, r.key(id, "*"), 10).Result()
		if err != nil {
			return err
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}

	return nil
}
