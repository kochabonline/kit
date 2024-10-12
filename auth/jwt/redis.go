package jwt

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type JwtRedis struct {
	jwt           *Jwt
	client        *redis.Client
	prefix        string
	accessPrefix  string
	refreshPrefix string
	multipleLogin bool
	expire        time.Duration
	refreshExpire time.Duration
}

type JwtRedisOption func(*JwtRedis)

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

func (r *JwtRedis) key(userId int64, key string) string {
	var builder strings.Builder
	builder.WriteString(r.prefix)
	builder.WriteString(":")
	builder.WriteString(strconv.FormatInt(userId, 10))
	builder.WriteString(":")
	builder.WriteString(key)
	return builder.String()
}

func (r *JwtRedis) getJti(claims jwt.MapClaims) string {
	if jti, ok := claims["jti"].(string); ok {
		return jti
	}

	return ""
}

func (r *JwtRedis) SetAccessJti(ctx context.Context, userId int64, jti string) error {
	if r.multipleLogin {
		return r.client.Set(ctx, r.key(userId, jti), "", r.expire).Err()
	}

	return r.client.Set(ctx, r.key(userId, r.accessPrefix), jti, r.expire).Err()
}

func (r *JwtRedis) SetRefreshJti(ctx context.Context, userId int64, jti string) error {
	if r.multipleLogin {
		return r.client.Set(ctx, r.key(userId, jti), "", r.refreshExpire).Err()
	}

	return r.client.Set(ctx, r.key(userId, r.refreshPrefix), jti, r.refreshExpire).Err()
}

func (r *JwtRedis) Get(ctx context.Context, userId int64, claims jwt.MapClaims) error {
	jti := r.getJti(claims)
	if jti == "" {
		return jwt.ErrTokenInvalidId
	}

	if r.multipleLogin {
		return r.client.Get(ctx, r.key(userId, jti)).Err()
	}

	return r.client.Get(ctx, r.key(userId, r.accessPrefix)).Err()

}

func (r *JwtRedis) Delete(ctx context.Context, userId int64) error {
	var cursor uint64
	var keys []string
	var err error
	for {
		var scanKeys []string
		scanKeys, cursor, err = r.client.Scan(ctx, cursor, r.key(userId, "*"), 100).Result()
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
