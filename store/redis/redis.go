package redis

import (
	"context"
	"runtime"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Client        *redis.Client
	ClusterClient *redis.ClusterClient
}

type Option func(*Redis)

func New(c *Config, opts ...Option) (*Redis, error) {
	_ = c.initConfig()
	r := &Redis{}

	for _, opt := range opts {
		opt(r)
	}

	return r.newClient(c)
}

func (r *Redis) newClient(c *Config) (*Redis, error) {
	if c.PoolSize == 0 {
		c.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	r.Client = redis.NewClient(&redis.Options{
		Addr:     c.Addr(),
		Password: c.Password,
		DB:       c.DB,
		Protocol: c.Protocol,
		PoolSize: c.PoolSize,
	})

	_, err := r.Client.Ping(context.Background()).Result()

	return r, err
}

func (r *Redis) Close() error {
	if r.Client == nil {
		return nil
	}

	return r.Client.Close()
}

func NewCluster(c *ClusterConfig, opts ...Option) (*Redis, error) {
	_ = c.initConfig()
	r := &Redis{}

	for _, opt := range opts {
		opt(r)
	}

	return r.newClusterClient(c)
}

func (r *Redis) newClusterClient(c *ClusterConfig) (*Redis, error) {
	if c.PoolSize == 0 {
		c.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	r.ClusterClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    c.Addrs,
		Password: c.Password,
		Protocol: c.Protocol,
		PoolSize: c.PoolSize,
	})

	_, err := r.ClusterClient.Ping(context.Background()).Result()

	return r, err
}

func (r *Redis) CloseCluster() error {
	if r.ClusterClient == nil {
		return nil
	}

	return r.ClusterClient.Close()
}
