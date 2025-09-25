package config

import (
	"testing"
)

type mock struct {
	Host    string  `json:"host"`
	Port    int     `json:"port" default:"8080"`
	Number  float64 `json:"number"`
	Enabled bool    `json:"enabled" default:"true"`
	Mock1   struct {
		Host string `json:"host" default:"localhost"`
	} `json:"mock1"`
	Apis []struct {
		Name    string            `json:"name"`
		Method  string            `json:"method" default:"GET"`
		Url     string            `json:"url"`
		Body    string            `json:"body"`
		Headers map[string]string `json:"headers"`
		Timeout int               `json:"timeout" default:"3"`
		Period  int               `json:"period" default:"10"`
	} `json:"apis"`
}

func TestConfig(t *testing.T) {
	cfg := new(mock)
	cfg.Port = 9090
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
