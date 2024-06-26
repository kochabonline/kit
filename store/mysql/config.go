package mysql

import (
	"fmt"
	"github.com/kochabonline/kit/core/reflect"
)

type Config struct {
	Host            string `json:"host" default:"localhost"`
	Port            int    `json:"port" default:"3306"`
	Username        string `json:"username" default:"root"`
	Password        string `json:"password"`
	DataBase        string `json:"database"`
	Charset         string `json:"charset" default:"utf8mb4"`
	ParseTime       *bool  `json:"parseTime" default:"true"`
	Loc             string `json:"loc" default:"Local"`
	Timeout         int    `json:"timeout" default:"10"`
	MaxIdleConns    int    `json:"maxIdleConns" default:"10"`
	MaxOpenConns    int    `json:"maxOpenConns" default:"100"`
	ConnMaxLifetime int    `jsin:"connMaxLifetime" default:"3600"`
	Level           string `json:"level" default:"silent"`
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c)
}

func (c *Config) dsn() string {
	parseTime := "False"
	if c.ParseTime != nil && *c.ParseTime {
		parseTime = "True"
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%s&loc=%s&timeout=%ds",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.DataBase,
		c.Charset,
		parseTime,
		c.Loc,
		c.Timeout)
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
