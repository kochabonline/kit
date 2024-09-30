package mysql

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Mysql struct {
	Client *gorm.DB
	config *Config
}

type Option func(*Mysql)

func New(c *Config, opts ...Option) (*Mysql, error) {
	m := &Mysql{
		config: c,
	}

	if err := m.config.init(); err != nil {
		return m, err
	}

	for _, opt := range opts {
		opt(m)
	}

	return m.new()
}

func (m *Mysql) new() (*Mysql, error) {
	client, err := gorm.Open(mysql.Open(m.config.dsn()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(m.config.LogLevel())),
	})
	if err != nil {
		return nil, err
	}
	m.Client = client

	sqlDB, err := m.Client.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(m.config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(m.config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(m.config.ConnMaxLifetime))

	return m, nil
}

func (m *Mysql) Close() error {
	if m.Client == nil {
		return nil
	}

	sqlDB, err := m.Client.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
