package mysql

import (
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Mysql struct {
	Client *gorm.DB
	once   sync.Once
}

type Option func(*Mysql)

func New(c *Config, opts ...Option) (*gorm.DB, error) {
	_ = c.initConfig()
	m := &Mysql{}

	for _, opt := range opts {
		opt(m)
	}

	return m.newClient(c)
}

func (m *Mysql) newClient(c *Config) (*gorm.DB, error) {
	var err error
	m.once.Do(func() {
		m.Client, err = gorm.Open(mysql.Open(c.dsn()), &gorm.Config{
			Logger: logger.Default.LogMode(logger.LogLevel(c.LogLevel())),
		})
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := m.Client.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(c.ConnMaxLifetime))

	return m.Client, nil
}

func CloseClient(client *gorm.DB) error {
	if client == nil {
		return nil
	}

	sqlDB, err := client.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
