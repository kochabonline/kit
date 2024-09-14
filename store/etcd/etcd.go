package etcd

import (
	"time"

	"go.etcd.io/etcd/client/v3"
)

type Etcd struct {
	Client *clientv3.Client
}

type Option func(*Etcd)

func New(c *Config, opts ...Option) (*Etcd, error) {
	e := &Etcd{}

	if err := c.initConfig(); err != nil {
		return e, err
	}

	for _, opt := range opts {
		opt(e)
	}

	return e.new(c)
}

func (e *Etcd) new(c *Config) (*Etcd, error) {
	var err error
	e.Client, err = clientv3.New(clientv3.Config{
		Endpoints:   c.Endpoints,
		Username:    c.Username,
		Password:    c.Password,
		DialTimeout: time.Duration(c.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Etcd) Close() error {
	if e.Client == nil {
		return nil
	}

	return e.Client.Close()
}
