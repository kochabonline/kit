package casbin

import (
	"github.com/casbin/govaluate"
	"github.com/kochabonline/kit/core/reflect"
	"gorm.io/gorm"
)

type Config struct {
	DB         *gorm.DB
	Model      string
	ExpireTime int `default:"60"`
	Function   map[string]govaluate.ExpressionFunction
}

func (c *Config) init() error {
	return reflect.SetDefaultTag(c)
}
