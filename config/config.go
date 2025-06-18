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

// Pre-defined environment key replacer to avoid repeated creation
var envKeyReplacer = strings.NewReplacer(".", "_")

type Config struct {
	Option Option `json:"option"`
	viper  *viper.Viper
}

type Option struct {
	Provider Provider // Provider is the provider of the configuration, e.g., file, etc.
	Path     []string // Path is the path to the configuration file, can be multiple paths.
	Name     string   // Name is the name of the configuration file without extension.
	Dest     any      // Dest is the destination where the configuration will be unmarshalled.
}

type ConfigOption func(*Option)

func WithProvider(provider Provider) ConfigOption {
	return func(o *Option) {
		o.Provider = provider
	}
}

func WithPath(path ...string) ConfigOption {
	return func(o *Option) {
		o.Path = path
	}
}

func WithName(name string) ConfigOption {
	return func(o *Option) {
		o.Name = name
	}
}

func WithDest(dest any) ConfigOption {
	return func(o *Option) {
		o.Dest = dest
	}
}

func New(opts ...ConfigOption) *Config {
	c := &Config{
		Option: Option{
			Provider: ProviderFile,
			Path:     []string{"."},
		},
		viper: viper.New(),
	}

	for _, opt := range opts {
		opt(&c.Option)
	}

	if err := c.init(); err != nil {
		log.Error().Err(err).Send()
		return nil
	}

	c.configureViper()

	return c
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c.Option.Dest)
}

// configureViper configures the default settings for the viper instance
func (c *Config) configureViper() {
	// Parse configuration file type
	extension := path.Ext(c.Option.Name)
	configType := strings.TrimPrefix(extension, ".")

	// Configure viper
	for _, configPath := range c.Option.Path {
		c.viper.AddConfigPath(configPath)
	}

	c.viper.SetConfigName(c.Option.Name)
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

	if err := c.viper.Unmarshal(&c.Option.Dest); err != nil {
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
