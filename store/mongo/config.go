package mongo

import (
	"strconv"
	"strings"

	"github.com/kochabonline/kit/core/reflect"
)

type Config struct {
	Host        string `json:"host" default:"localhost"`
	Port        int    `json:"port" default:"27017"`
	User        string `json:"user" default:"root"`
	Password    string `json:"password"`
	MaxPoolSize int    `json:"maxPoolSize" default:"10"`
	Timeout     int    `json:"timeout" default:"3"`
}

func (c *Config) uri() string {
	var builer strings.Builder

	builer.WriteString("mongodb://")
	if c.Password != "" {
		builer.WriteString(c.User)
		builer.WriteString(":")
		builer.WriteString(c.Password)
	}
	builer.WriteString("@")
	builer.WriteString(c.Host)
	builer.WriteString(":")
	builer.WriteString(strconv.Itoa(c.Port))
	builer.WriteString("/?maxPoolSize=")
	builer.WriteString(strconv.Itoa(c.MaxPoolSize))

	return builer.String()
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c)
}
