package casbin

import (
	"github.com/kochabonline/kit/core/reflect"
	"gorm.io/gorm"
)

type Config struct {
	DB         *gorm.DB
	Model      string
	ExpireTime int `default:"60"`
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c)
}
