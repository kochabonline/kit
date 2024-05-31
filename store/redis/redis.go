package redis

import (
	"context"
	"runtime"

	"github.com/redis/go-redis/v9"
)

type Single struct {
	Client *redis.Client
}

type Cluster struct {
	Client *redis.ClusterClient
}

type SingleOption func(*Single)

func NewClient(c *Config, opts ...SingleOption) (*redis.Client, error) {
	_ = c.initConfig()
	s := &Single{}

	for _, opt := range opts {
		opt(s)
	}

	return s.newClient(c)
}

func (s *Single) newClient(c *Config) (*redis.Client, error) {
	if c.PoolSize == 0 {
		c.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     c.Addr(),
		Password: c.Password,
		DB:       c.DB,
		Protocol: c.Protocol,
		PoolSize: c.PoolSize,
	})

	_, err := client.Ping(context.Background()).Result()

	return client, err
}

func CloseClient(client *redis.Client) error {
	if client == nil {
		return nil
	}

	return client.Close()
}

type ClusterOption func(*Cluster)

func NewClusterClient(c *ClusterConfig, opts ...ClusterOption) (*redis.ClusterClient, error) {
	_ = c.initConfig()
	cl := &Cluster{}

	for _, opt := range opts {
		opt(cl)
	}

	return cl.newClusterClient(c)
}

func (cl *Cluster) newClusterClient(c *ClusterConfig) (*redis.ClusterClient, error) {
	if c.PoolSize == 0 {
		c.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    c.Addrs,
		Password: c.Password,
		Protocol: c.Protocol,
		PoolSize: c.PoolSize,
	})

	_, err := client.Ping(context.Background()).Result()

	return client, err
}

func CloseClusterClient(client *redis.ClusterClient) error {
	if client == nil {
		return nil
	}

	return client.Close()
}
