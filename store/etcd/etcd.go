package etcd

import (
	"time"

	"go.etcd.io/etcd/client/v3"
)

type Etcd struct {
	Client *clientv3.Client
	config *Config
}

type Option func(*Etcd)

func New(c *Config, opts ...Option) (*Etcd, error) {
	e := &Etcd{
		config: c,
	}

	if err := e.config.init(); err != nil {
		return e, err
	}

	for _, opt := range opts {
		opt(e)
	}

	return e.new()
}

func (e *Etcd) new() (*Etcd, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   e.config.Endpoints,
		Username:    e.config.Username,
		Password:    e.config.Password,
		DialTimeout: time.Duration(e.config.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	e.Client = client

	return e, nil
}

func (e *Etcd) Close() error {
	if e.Client == nil {
		return nil
	}

	return e.Client.Close()
}
