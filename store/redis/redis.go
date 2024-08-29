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

func NewClient(c *Config, opts ...SingleOption) (*Single, error) {
	if err := c.initConfig(); err != nil {
		return nil, err
	}

	s := &Single{}

	for _, opt := range opts {
		opt(s)
	}

	return s.newClient(c)
}

func (s *Single) newClient(c *Config) (*Single, error) {
	if c.PoolSize == 0 {
		c.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	s.Client = redis.NewClient(&redis.Options{
		Addr:     c.Addr(),
		Password: c.Password,
		DB:       c.DB,
		Protocol: c.Protocol,
		PoolSize: c.PoolSize,
	})

	_, err := s.Client.Ping(context.Background()).Result()

	return s, err
}

func (s *Single) Close() error {
	if s.Client == nil {
		return nil
	}
	
	return s.Client.Close()
}

type ClusterOption func(*Cluster)

func NewClusterClient(c *ClusterConfig, opts ...ClusterOption) (*Cluster, error) {
	_ = c.initConfig()
	cl := &Cluster{}

	for _, opt := range opts {
		opt(cl)
	}

	return cl.newClusterClient(c)
}

func (cl *Cluster) newClusterClient(c *ClusterConfig) (*Cluster, error) {
	if c.PoolSize == 0 {
		c.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	cl.Client = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    c.Addrs,
		Password: c.Password,
		Protocol: c.Protocol,
		PoolSize: c.PoolSize,
	})

	_, err := cl.Client.Ping(context.Background()).Result()

	return cl, err
}

func (cl *Cluster) Close() error {
	if cl.Client == nil {
		return nil
	}

	return cl.Client.Close()
}
