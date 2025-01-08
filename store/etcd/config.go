package etcd

import "github.com/kochabonline/kit/core/reflect"

type Config struct {
	Endpoints   []string `json:"endpoints" default:"localhost:2379"`
	Username    string   `json:"username" default:"root"`
	Password    string   `json:"password"`
	DialTimeout int64    `json:"dialTimeout" default:"5"`
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c)
}
