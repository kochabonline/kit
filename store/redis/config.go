package redis

import (
	"strconv"
	"strings"

	"github.com/kochabonline/kit/core/reflect"
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
	Addrs    []string `json:"addrs" default:"localhost:6379"`
	Password string   `json:"password"`
	DB       int      `json:"db" default:"0"`
	Protocol int      `json:"protocol" default:"3"`
	PoolSize int      `json:"poolSize"`
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c)
}

func (c *Config) Addr() string {
	var builder strings.Builder
	builder.WriteString(c.Host)
	builder.WriteString(":")
	builder.WriteString(strconv.Itoa(c.Port))
	return builder.String()
}

func (c *ClusterConfig) init() error {
	return reflect.SetDefaultTag(c)
}
