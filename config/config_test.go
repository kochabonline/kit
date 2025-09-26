package config

import (
	"testing"
)

type api struct {
	Name    string            `json:"name"`
	Method  string            `json:"method" default:"GET"`
	Url     string            `json:"url"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout" default:"3"`
	Period  int               `json:"period" default:"10"`
}

type server struct {
	Host string `json:"host" default:"localhost"`
}

type mock struct {
	Host    string  `json:"host" validate:"required,min=2,max=10"`
	Port    int     `json:"port" default:"8080"`
	Number  float64 `json:"number" default:"1.23"`
	Enabled bool    `json:"enabled" default:"true"`
	Server  server  `json:"server"`
	Apis    []api   `json:"apis"`
}

func TestConfig(t *testing.T) {
	cfg := new(mock)
	cfg.Port = 9090
	cfg.Host = "12"
	c := New(WithTarget(cfg))

	if err := c.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
	if err := c.WatchConfig(); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)
}
