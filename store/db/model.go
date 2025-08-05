package db

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kochabonline/kit/core/reflect"
)

// 数据库驱动类型
type Driver string

const (
	DriverMySQL      Driver = "mysql"
	DriverPostgreSQL Driver = "postgres"
	DriverSQLite     Driver = "sqlite"
)

// orm 类型
type Orm string

const (
	OrmGorm Orm = "gorm"
	OrmEnt  Orm = "ent"
)

// Config 数据库配置结构体
type Config struct {
	Driver       Driver       `json:"driver" default:"mysql"`
	Orm          Orm          `json:"orm" default:"gorm"`
	DriverConfig DriverConfig `json:"driverConfig"`
}

// DriverConfig 接口定义
// 用于不同数据库驱动的配置实现
type DriverConfig interface {
	init() error
	Dsn() string
	LogLevel() int
}

// Mysql 配置
type MysqlConfig struct {
	Host            string `json:"host" default:"localhost"`
	Port            int    `json:"port" default:"3306"`
	User            string `json:"user" default:"root"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	Charset         string `json:"charset" default:"utf8mb4"`
	ParseTime       bool   `json:"parseTime" default:"true"`
	Loc             string `json:"loc" default:"Local"`
	Timeout         int    `json:"timeout" default:"10"`
	MaxIdleConns    int    `json:"maxIdleConns" default:"10"`
	MaxOpenConns    int    `json:"maxOpenConns" default:"100"`
	ConnMaxLifetime int    `json:"connMaxLifetime" default:"3600"`
	Level           string `json:"level" default:"silent"`
}

func (c *MysqlConfig) init() error {
	return reflect.SetDefaultTag(c)
}

func (c *MysqlConfig) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&timeout=%ds&maxIdleConns=%d&maxOpenConns=%d&connMaxLifetime=%ds&level=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Charset,
		c.ParseTime,
		c.Loc,
		c.Timeout,
		c.MaxIdleConns,
		c.MaxOpenConns,
		c.ConnMaxLifetime,
		strings.ToLower(c.Level),
	)
}

func (c *MysqlConfig) LogLevel() int {
	switch strings.ToLower(c.Level) {
	case "silent":
		return 0
	case "error":
		return 1
	case "warn":
		return 2
	case "info":
		return 3
	default:
		return 0 // 默认返回 silent
	}
}

// PostgresConfig 配置
type PostgresConfig struct {
	Host            string `json:"host" default:"localhost"`
	Port            int    `json:"port" default:"5432"`
	User            string `json:"user" default:"postgres"`
	Password        string `json:"password"`
	Database        string `json:"database"`
	SSLMode         string `json:"sslmode" default:"disable"`
	ConnectTimeout  int    `json:"connectTimeout" default:"10"`
	MaxIdleConns    int    `json:"maxIdleConns" default:"10"`
	MaxOpenConns    int    `json:"maxOpenConns" default:"100"`
	ConnMaxLifetime int    `json:"connMaxLifetime" default:"3600"`
	Level           string `json:"level" default:"silent"`
}

func (c *PostgresConfig) init() error {
	return reflect.SetDefaultTag(c)
}

func (c *PostgresConfig) Dsn() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d max_idle_conns=%d max_open_conns=%d conn_max_lifetime=%ds level=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Database,
		c.SSLMode,
		c.ConnectTimeout,
		c.MaxIdleConns,
		c.MaxOpenConns,
		c.ConnMaxLifetime,
		strings.ToLower(c.Level),
	)
}

func (c *PostgresConfig) LogLevel() int {
	switch strings.ToLower(c.Level) {
	case "silent":
		return 0
	case "error":
		return 1
	case "warn":
		return 2
	case "info":
		return 3
	default:
		return 0 // 默认返回 silent
	}
}

// SQLiteConfig 配置
type SQLiteConfig struct {
	FilePath        string `json:"filePath" default:"./data.db"`
	CacheSize       int    `json:"cacheSize" default:"10000"`
	BusyTimeout     int    `json:"busyTimeout" default:"5000"`
	SyncMode        string `json:"syncMode" default:"normal"`
	ForeignKeys     bool   `json:"foreignKeys" default:"true"`
	CacheMode       string `json:"cacheMode" default:"default"`
	MaxIdleConns    int    `json:"maxIdleConns" default:"10"`
	MaxOpenConns    int    `json:"maxOpenConns" default:"100"`
	ConnMaxLifetime int    `json:"connMaxLifetime" default:"3600"`
	Level           string `json:"level" default:"silent"`
}

func (c *SQLiteConfig) init() error {
	return reflect.SetDefaultTag(c)
}

func (c *SQLiteConfig) Dsn() string {
	return fmt.Sprintf("file:%s?cache=%s&mode=rw&_busy_timeout=%d&_sync=%s&_foreign_keys=%t&_cache=%s&max_idle_conns=%d&max_open_conns=%d&conn_max_lifetime=%ds&level=%s",
		c.FilePath,
		strconv.Itoa(c.CacheSize),
		c.BusyTimeout,
		c.SyncMode,
		c.ForeignKeys,
		c.CacheMode,
		c.MaxIdleConns,
		c.MaxOpenConns,
		c.ConnMaxLifetime,
		strings.ToLower(c.Level),
	)
}

func (c *SQLiteConfig) LogLevel() int {
	switch strings.ToLower(c.Level) {
	case "silent":
		return 0
	case "error":
		return 1
	case "warn":
		return 2
	case "info":
		return 3
	default:
		return 0 // 默认返回 silent
	}
}
