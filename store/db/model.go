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

// Config 数据库配置结构体
type Config struct {
	Driver       Driver       `json:"driver" default:"mysql"`
	DriverConfig DriverConfig `json:"driverConfig"`
}

// DriverConfig 接口定义
// 用于不同数据库驱动的配置实现
type DriverConfig interface {
	Init() error
	Dsn() string
	Level() int
	CloneConn() *Connection
}

// Connection 连接池配置
type Connection struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int
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
	LogLevel        string `json:"logLevel" default:"silent"`
}

func (c *MysqlConfig) Init() error {
	return reflect.SetDefaultTag(c)
}

func (c *MysqlConfig) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s&timeout=%ds",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Charset,
		c.ParseTime,
		c.Loc,
		c.Timeout,
	)
}

func (c *MysqlConfig) Level() int {
	switch strings.ToLower(c.LogLevel) {
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

func (c *MysqlConfig) CloneConn() *Connection {
	return &Connection{
		MaxIdleConns:    c.MaxIdleConns,
		MaxOpenConns:    c.MaxOpenConns,
		ConnMaxLifetime: c.ConnMaxLifetime,
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
	LogLevel        string `json:"logLevel" default:"silent"`
}

func (c *PostgresConfig) Init() error {
	return reflect.SetDefaultTag(c)
}

func (c *PostgresConfig) Dsn() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Database,
		c.SSLMode,
		c.ConnectTimeout,
	)
}

func (c *PostgresConfig) Level() int {
	switch strings.ToLower(c.LogLevel) {
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

func (c *PostgresConfig) CloneConn() *Connection {
	return &Connection{
		MaxIdleConns:    c.MaxIdleConns,
		MaxOpenConns:    c.MaxOpenConns,
		ConnMaxLifetime: c.ConnMaxLifetime,
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
	LogLevel        string `json:"logLevel" default:"silent"`
}

func (c *SQLiteConfig) Init() error {
	return reflect.SetDefaultTag(c)
}

func (c *SQLiteConfig) Dsn() string {
	return fmt.Sprintf("file:%s?cache=%s&mode=rw&_busy_timeout=%d&_sync=%s&_foreign_keys=%t&_cache=%s",
		c.FilePath,
		strconv.Itoa(c.CacheSize),
		c.BusyTimeout,
		c.SyncMode,
		c.ForeignKeys,
		c.CacheMode,
	)
}

func (c *SQLiteConfig) Level() int {
	switch strings.ToLower(c.LogLevel) {
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

func (c *SQLiteConfig) CloneConn() *Connection {
	return &Connection{
		MaxIdleConns:    c.MaxIdleConns,
		MaxOpenConns:    c.MaxOpenConns,
		ConnMaxLifetime: c.ConnMaxLifetime,
	}
}
