package mysql

import (
	"time"

	"github.com/kochabonline/kit/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Mysql struct {
	Client *gorm.DB
}

type Option func(*Mysql)

func New(c *Config, opts ...Option) (*Mysql, error) {
	err := c.initConfig()
	if err != nil {
		return nil, err
	}

	m := &Mysql{}

	for _, opt := range opts {
		opt(m)
	}

	return m.new(c)
}

func (m *Mysql) new(c *Config) (*Mysql, error) {
	var err error
	m.Client, err = gorm.Open(mysql.Open(c.dsn()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(c.LogLevel())),
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
	log.Info("Closing MySQL connection")
	return sqlDB.Close()
}
