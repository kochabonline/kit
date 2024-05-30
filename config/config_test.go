package config

import (
	"testing"
	"time"
)

type mock struct {
	Host string `json:"host" default:"localhost"`
	Port int    `json:"port" default:"8080"`
	User string `json:"user" default:"root"`
	Pass string `json:"pass" default:"root"`
}

func TestConfig(t *testing.T) {
	cfg := new(mock)
	c := NewConfig(Option{
		Provider: ProviderFile,
		Target:   cfg,
	})

	if err := c.Read(); err != nil {
		t.Error(err)
	}

	if err := c.Watch(); err != nil {
		t.Error(err)
	}

	time.Sleep(10 * time.Second)
	t.Log(cfg)
}
