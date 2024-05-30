package redis

import (
	"fmt"

	"github.com/kochabonline/kit/reflect"
)

type Config struct {
	Host     string `json:"host" default:"localhost"`
	Port     int    `json:"port" default:"6379"`
	Password string `json:"password"`
	DB       int    `json:"db" default:"0"`
	Protocol int    `json:"protocol" default:"3"`
	PoolSize int    `json:"poolSize"`
}

type ClusterConfig struct {
	Addrs    []string `json:"addrs"`
	Password string   `json:"password"`
	DB       int      `json:"db" default:"0"`
	Protocol int      `json:"protocol" default:"3"`
	PoolSize int      `json:"poolSize"`
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c)
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *ClusterConfig) initConfig() error {
	return reflect.SetDefaultTag(c)
}
