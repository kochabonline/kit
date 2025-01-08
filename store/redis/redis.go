package redis

import (
	"context"
	"runtime"

	"github.com/redis/go-redis/v9"
)

type Single struct {
	Client *redis.Client
	config *Config
}

type Cluster struct {
	Client *redis.ClusterClient
	config *ClusterConfig
}

type SingleOption func(*Single)

func NewClient(c *Config, opts ...SingleOption) (*Single, error) {
	s := &Single{
		config: c,
	}

	if err := s.config.init(); err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(s)
	}

	return s.newClient()
}

func (s *Single) newClient() (*Single, error) {
	if s.config.PoolSize == 0 {
		s.config.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	s.Client = redis.NewClient(&redis.Options{
		Addr:     s.config.Addr(),
		Password: s.config.Password,
		DB:       s.config.DB,
		Protocol: s.config.Protocol,
		PoolSize: s.config.PoolSize,
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
	cl := &Cluster{
		config: c,
	}

	if err := cl.config.init(); err != nil {
		return cl, err
	}

	for _, opt := range opts {
		opt(cl)
	}

	return cl.newClusterClient()
}

func (cl *Cluster) newClusterClient() (*Cluster, error) {
	if cl.config.PoolSize == 0 {
		cl.config.PoolSize = 10 * runtime.GOMAXPROCS(0)
	}

	cl.Client = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cl.config.Addrs,
		Password: cl.config.Password,
		Protocol: cl.config.Protocol,
		PoolSize: cl.config.PoolSize,
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
