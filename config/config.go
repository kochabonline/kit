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
	Option Option `json:"option"`
	log    *log.Helper
}

type Option struct {
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

func NewConfig(cf Option, opts ...ConfigOption) *Config {
	c := &Config{
		Option: cf,
		log:    log.DefaultLogger,
	}
	_ = c.initConfig()

	for _, opt := range opts {
		opt(c)
	}

	c.viper()

	return c
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c.Option.Target)
}

func (c *Config) viper() {
	if c.Option.Path == "" {
		c.Option.Path = "."
	}
	extension := path.Ext(c.Option.Name)
	configType := strings.TrimPrefix(extension, ".")

	viper.AddConfigPath(c.Option.Path)
	viper.SetConfigName(c.Option.Name)
	viper.SetConfigType(configType)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}

func (c *Config) Read() error {
	if err := viper.ReadInConfig(); err != nil {
		return errors.Internal("config read error", err.Error())
	}

	if err := viper.Unmarshal(&c.Option.Target); err != nil {
		return errors.Internal("config unmarshal error", err.Error())
	}

	return nil
}

func (c *Config) Watch() error {
	ch := make(chan error, 1)

	viper.OnConfigChange(func(e fsnotify.Event) {
		c.log.Infof("Config file changed: %s", e.Name)
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
