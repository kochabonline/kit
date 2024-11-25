package scheduler

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type DistributedLock struct {
	client *redis.Client
}

func NewDistributedLock(client *redis.Client) *DistributedLock {
	return &DistributedLock{
		client: client,
	}
}

func (dl *DistributedLock) TryLock(ctx context.Context, key string, value string, ttl time.Duration) bool {
	success, err := dl.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false
	}
	return success
}

func (dl *DistributedLock) Unlock(ctx context.Context, key string, value string) {
	storedValue, err := dl.client.Get(ctx, key).Result()
	if err == redis.Nil || storedValue != value {
		return
	}
	dl.client.Del(ctx, key)
}
