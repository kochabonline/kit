package config

import (
	"testing"
)

type mock struct {
	Host    string  `json:"host"`
	Port    int     `json:"port" default:"8080"`
	Number  float64 `json:"number"`
	Enabled bool    `json:"enabled"`
	Mock1   struct {
		Host string `json:"host" default:"localhost"`
	}
}

func TestConfig(t *testing.T) {
	cfg := new(mock)
	c := NewConfig(Option{
		Target: cfg,
	})

	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
	if err := c.WatchConfig(); err != nil {
		t.Fatal(err)
	}

	t.Log(cfg)
}
