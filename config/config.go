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

var (
	envKeyReplacer = strings.NewReplacer(".", "_")
)

type Config struct {
	viper    *viper.Viper // viper is the underlying viper instance
	provider Provider     // provider is the provider of the configuration, e.g., file, etc.
	path     []string     // path is the path to the configuration file, can be multiple paths.
	name     string       // name is the name of the configuration file without extension.
	dest     any          // dest is the destination where the configuration will be unmarshalled.
}

type Option func(*Config)

func WithViper(v *viper.Viper) Option {
	return func(c *Config) {
		c.viper = v
	}
}

func WithProvider(provider Provider) Option {
	return func(c *Config) {
		c.provider = provider
	}
}

func WithPath(path ...string) Option {
	return func(c *Config) {
		c.path = path
	}
}

func WithName(name string) Option {
	return func(c *Config) {
		c.name = name
	}
}

func WithDest(dest any) Option {
	return func(c *Config) {
		c.dest = dest
	}
}

func New(opts ...Option) *Config {
	c := &Config{
		provider: ProviderFile,
		path:     []string{"."},
		viper:    viper.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.init(); err != nil {
		log.Fatal().Err(err).Send()
		return nil
	}

	c.setupViper()

	return c
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c.dest)
}

// setupViper configures the viper instance based on the Config settings
func (c *Config) setupViper() {
	// Determine config type from file extension
	extension := path.Ext(c.name)
	configType := strings.TrimPrefix(extension, ".")

	// Add configuration paths to viper
	for _, configPath := range c.path {
		c.viper.AddConfigPath(configPath)
	}

	c.viper.SetConfigName(c.name)
	c.viper.SetConfigType(configType)
	c.viper.AutomaticEnv()
	c.viper.SetEnvKeyReplacer(envKeyReplacer)
}

func (c *Config) GetViper() *viper.Viper {
	return c.viper
}

func (c *Config) ReadInConfig() error {
	if err := c.viper.ReadInConfig(); err != nil {
		return err
	}

	if err := c.viper.Unmarshal(&c.dest); err != nil {
		return err
	}

	return nil
}

func (c *Config) WatchConfig() error {
	c.viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info().Msgf("config file changed: %s", e.Name)
		if err := c.ReadInConfig(); err != nil {
			log.Error().Err(err).Msg("failed to reload config")
		}
	})
	c.viper.WatchConfig()
	return nil
}
