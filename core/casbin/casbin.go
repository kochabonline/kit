package casbin

import (
	"errors"
	"time"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	_ "github.com/go-sql-driver/mysql"
)

type Casbin struct {
	SyncedCachedEnforcer *casbin.SyncedCachedEnforcer
}

func New(config Config) (*Casbin, error) {
	if config.DB == nil || config.Model == "" {
		return nil, errors.New("invalid casbin config")
	}

	c := &Casbin{}
	if err := config.init(); err != nil {
		return nil, err
	}

	a, err := gormadapter.NewAdapterByDB(config.DB)
	if err != nil {
		return nil, err
	}

	c.SyncedCachedEnforcer, err = casbin.NewSyncedCachedEnforcer(config.Model, a)
	if err != nil {
		return nil, err
	}
	c.SyncedCachedEnforcer.SetExpireTime(time.Duration(config.ExpireTime) * time.Second)
	if err := c.SyncedCachedEnforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	if config.Function != nil {
		for k, v := range config.Function {
			c.SyncedCachedEnforcer.AddFunction(k, v)
		}
	}

	return c, nil
}

func (c *Casbin) Close() {
	if c.SyncedCachedEnforcer != nil {
		c.SyncedCachedEnforcer.StopAutoLoadPolicy()
		c.SyncedCachedEnforcer.ClearPolicy()
	}
}
