package casbin

import (
	"errors"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	_ "github.com/go-sql-driver/mysql"
)

type Casbin struct {
	E *casbin.SyncedCachedEnforcer
}

func New(config Config) (*Casbin, error) {
	if config.DB == nil || config.Model == "" {
		return nil, errors.New("invalid casbin config")
	}

	c := &Casbin{}
	a, err := gormadapter.NewAdapterByDB(config.DB)
	if err != nil {
		return nil, err
	}

	c.E, err = casbin.NewSyncedCachedEnforcer(config.Model, a)
	if err != nil {
		return nil, err
	}

	if err := c.E.LoadPolicy(); err != nil {
		return nil, err
	}

	return c, nil
}
