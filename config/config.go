package config

import (
	"path"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"github.com/kochabonline/kit/core/reflect"
	"github.com/kochabonline/kit/errors"
	"github.com/kochabonline/kit/log"
)

type Interface interface {
	Read() error
	Watch() error
}

type Config struct {
	Options Options `json:"option"`
}

type Options struct {
	// Provider is the type of configuration provider.
	Provider Provider
	// Path is the path to the configuration file.
	Path string
	// Name is the name of the configuration file.
	Name string
	// Target is the target of the configuration.
	Target any
}

type ConfigOption func(*Config)

func NewConfig(cfgOptions Options, opts ...ConfigOption) *Config {
	c := &Config{
		Options: cfgOptions,
	}
	_ = c.initConfig()

	for _, opt := range opts {
		opt(c)
	}

	c.viper()

	return c
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c.Options.Target)
}

func (c *Config) viper() {
	if c.Options.Path == "" {
		c.Options.Path = "."
	}
	extension := path.Ext(c.Options.Name)
	configType := strings.TrimPrefix(extension, ".")

	viper.AddConfigPath(c.Options.Path)
	viper.SetConfigName(c.Options.Name)
	viper.SetConfigType(configType)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}

func (c *Config) Read() error {
	if err := viper.ReadInConfig(); err != nil {
		return errors.Internal("config read error", err.Error())
	}

	if err := viper.Unmarshal(&c.Options.Target); err != nil {
		return errors.Internal("config unmarshal error", err.Error())
	}

	return nil
}

func (c *Config) Watch() error {
	ch := make(chan error, 1)

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("Config file changed: %s", e.Name)
		if err := c.Read(); err != nil {
			ch <- err
		}
	})

	viper.WatchConfig()

	select {
	case err := <-ch:
		return err
	default:
		return nil
	}
}
