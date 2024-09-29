package rate

import (
	"context"
	_ "embed"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	//go:embed tokenbucket.lua
	tokenBucketLua       string
	tokenBucketLuaScript = redis.NewScript(tokenBucketLua)
)

type TokenBucketLimiter struct {
	client    *redis.Client
	bucketKey string
	capacity  int
	rate      int
	script    *redis.Script
}

func NewTokenBucketLimiter(client *redis.Client, bucketKey string, capacity, rate int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client:    client,
		bucketKey: bucketKey,
		capacity:  capacity,
		rate:      rate,
		script:    tokenBucketLuaScript,
	}
}

func (l *TokenBucketLimiter) Allow() bool {
	return l.AllowN(time.Now(), 1)
}

func (l *TokenBucketLimiter) AllowN(t time.Time, n int) bool {
	return l.AllowNCtx(context.Background(), t, n)
}

func (l *TokenBucketLimiter) AllowNCtx(ctx context.Context, t time.Time, n int) bool {
	return l.reserveN(ctx, t, n)
}

func (l *TokenBucketLimiter) reserveN(ctx context.Context, t time.Time, n int) bool {
	now := t.Unix()
	result, err := l.script.Run(ctx, l.client, []string{l.bucketKey}, l.capacity, l.rate, now, n).Result()
	if err != nil {
		return false
	}

	return result.(int64) == 1
}
