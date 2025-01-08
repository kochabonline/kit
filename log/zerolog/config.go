package zerolog

import (
	"path"
	"strings"

	"github.com/kochabonline/kit/core/reflect"
)

type Mode int

const (
	ModeTime Mode = iota
	ModeSize
)

type Config struct {
	Mode             Mode   `default:"0"`
	Filepath         string `default:"log"`
	Filename         string `default:"app"`
	FileExt          string `default:"log"`
	RotatelogsConfig RotatelogsConfig
	LumberjackConfig LumberjackConfig
}

type RotatelogsConfig struct {
	MaxAge       int `default:"24"`
	RotationTime int `default:"1"`
}

type LumberjackConfig struct {
	MaxSize    int  `default:"100"`
	MaxBackups int  `default:"5"`
	MaxAge     int  `default:"30"`
	Compress   bool `default:"false"`
}

func (c *Config) fileFullPath() string {
	return c.fileFullPathWithDataTimeFormat("")
}

func (c *Config) fileFullPathWithDataTimeFormat(dataTimeFormat string) string {
	var builder strings.Builder
	builder.WriteString(c.Filename)
	builder.WriteString(".")
	if dataTimeFormat != "" {
		builder.WriteString(dataTimeFormat)
		builder.WriteString(".")
	}
	builder.WriteString(c.FileExt)

	return path.Join(c.Filepath, builder.String())
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c)
}
