package mysql

import (
	"strconv"
	"strings"

	"github.com/kochabonline/kit/core/reflect"
)

type Config struct {
	Host            string `json:"host" default:"localhost"`
	Port            int    `json:"port" default:"3306"`
	User            string `json:"user" default:"root"`
	Password        string `json:"password"`
	DataBase        string `json:"database"`
	Charset         string `json:"charset" default:"utf8mb4"`
	ParseTime       bool   `json:"parseTime" default:"true"`
	Loc             string `json:"loc" default:"Local"`
	Timeout         int    `json:"timeout" default:"10"`
	MaxIdleConns    int    `json:"maxIdleConns" default:"10"`
	MaxOpenConns    int    `json:"maxOpenConns" default:"100"`
	ConnMaxLifetime int    `jsin:"connMaxLifetime" default:"3600"`
	Level           string `json:"level" default:"silent"`
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c)
}

func (c *Config) dsn() string {
	var builder strings.Builder

	builder.WriteString(c.User)
	builder.WriteString(":")
	builder.WriteString(c.Password)
	builder.WriteString("@tcp(")
	builder.WriteString(c.Host)
	builder.WriteString(":")
	builder.WriteString(strconv.Itoa(c.Port))
	builder.WriteString(")/")
	builder.WriteString(c.DataBase)
	builder.WriteString("?charset=")
	builder.WriteString(c.Charset)
	builder.WriteString("&loc=")
	builder.WriteString(c.Loc)
	builder.WriteString("&timeout=")
	builder.WriteString(strconv.Itoa(c.Timeout))
	builder.WriteString("s")
	if c.ParseTime {
		builder.WriteString("&parseTime=True")
	}

	return builder.String()
}

func (c *Config) LogLevel() int {
	switch c.Level {
	case "silent":
		return 1
	case "error":
		return 2
	case "warn":
		return 3
	case "info":
		return 4
	default:
		return 1
	}
}
