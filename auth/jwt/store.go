package jwt

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type JwtRedisStore struct {
	client *redis.Client
	prefix string
}

func NewJwtRedisStore(client *redis.Client) *JwtRedisStore {
	token := &JwtRedisStore{
		client: client,
		prefix: "jwt",
	}

	return token
}

func (r *JwtRedisStore) key(key string) string {
	var builder strings.Builder
	builder.WriteString(r.prefix)
	builder.WriteString(":")
	builder.WriteString(key)
	return builder.String()
}

func (r *JwtRedisStore) Set(ctx context.Context, key string, value any, expiration int64) error {
	return r.client.Set(ctx, r.key(key), value, time.Duration(expiration)*time.Second).Err()
}

func (r *JwtRedisStore) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, r.key(key)).Result()
}

func (r *JwtRedisStore) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.key(key)).Err()
}
