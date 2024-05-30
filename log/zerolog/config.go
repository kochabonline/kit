package zerolog

import "github.com/kochabonline/kit/reflect"

type Mode int

const (
	ModeTime Mode = iota
	ModeSize
)

type Config struct {
	Mode         Mode   `default:"0"`
	Filepath     string `default:"log"`
	Filename     string `default:"app.log"`
	MaxSize      int    `default:"100"`
	MaxBackups   int    `default:"5"`
	MaxAge       int    `default:"30"`
	Compress     bool   `default:"false"`
	MaxAgeDay    int    `default:"7"`
	RotationTime int    `default:"1"`
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c)
}
