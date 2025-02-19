package config

import (
	"path"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"github.com/kochabonline/kit/core/reflect"
	"github.com/kochabonline/kit/log"
)

type Provider int

const (
	ProviderFile Provider = iota
)

type Config struct {
	Option Option `json:"option"`
	log    log.Helper
}

type Option struct {
	Provider Provider // Provider is the type of configuration provider.
	Path     []string // Path is the path to the configuration file.
	Name     string   // Name is the name of the configuration file.
	Target   any      // Target is the target of the configuration.
}

type ConfigOption func(*Config)

func WithLogger(log log.Helper) ConfigOption {
	return func(c *Config) {
		c.log = log
	}
}

func New(option Option, opts ...ConfigOption) *Config {
	c := &Config{
		Option: option,
		log:    log.DefaultLogger,
	}

	if err := c.init(); err != nil {
		c.log.Fatal(err)
	}

	for _, opt := range opts {
		opt(c)
	}

	c.setDefault()

	return c
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c.Option.Target)
}

func (c *Config) setDefault() {
	if c.Option.Path == nil {
		c.Option.Path = []string{"."}
	}
	extension := path.Ext(c.Option.Name)
	configType := strings.TrimPrefix(extension, ".")

	for _, path := range c.Option.Path {
		viper.AddConfigPath(path)
	}
	viper.SetConfigName(c.Option.Name)
	viper.SetConfigType(configType)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}

func (c *Config) GetViper() *viper.Viper {
	return viper.GetViper()
}

func (c *Config) ReadInConfig() error {
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := viper.Unmarshal(&c.Option.Target); err != nil {
		return err
	}

	return nil
}

func (c *Config) WatchConfig() error {
	ch := make(chan error, 1)

	viper.OnConfigChange(func(e fsnotify.Event) {
		c.log.Infof("config file changed: %s", e.Name)
		if err := c.ReadInConfig(); err != nil {
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
