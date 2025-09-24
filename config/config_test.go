package config

import (
	"testing"
)

type mock struct {
	Host    string  `json:"host" validate:"required,min=2"`
	Port    int     `json:"port" default:"8080"`
	Number  float64 `json:"number"`
	Enabled bool    `json:"enabled" default:"true"`
	Mock1   struct {
		Host string `json:"host" default:"localhost"`
	}
}

func TestConfig(t *testing.T) {
	cfg := new(mock)
	cfg.Host = "1"
	c := New(WithDest(cfg))

	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
	if err := c.WatchConfig(); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
}
