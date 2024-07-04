package config

import (
	"testing"
)

type mock struct {
	Host    string `json:"host" default:"localhost"`
	Port    int    `json:"port" default:"8080"`
	Number  float64 `json:"number"`
	Enabled bool   `json:"enabled"`
	Mock1   struct {
		Host string `json:"host" default:"localhost"`
	}
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

	t.Log(cfg)
}
