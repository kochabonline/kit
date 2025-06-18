package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/kochabonline/kit/core/reflect"
)

// 日志轮转模式
type RotateMode int

const (
	RotateModeTime RotateMode = iota
	RotateModeSize
)

type Config struct {
	RotateMode       RotateMode
	Filepath         string `default:"log"`
	Filename         string `default:"app"`
	FileExt          string `default:"log"`
	RotatelogsConfig RotatelogsConfig
	LumberjackConfig LumberjackConfig
}

type RotatelogsConfig struct {
	MaxAge       int `default:"24"`
	RotationTime int `default:"1"`
}

type LumberjackConfig struct {
	MaxSize    int  `default:"100"`
	MaxBackups int  `default:"5"`
	MaxAge     int  `default:"30"`
	Compress   bool `default:"false"`
}

type Logger struct {
	zerolog.Logger
}

var (
	DefaultLogger *Logger
	consoleWriter zerolog.ConsoleWriter
)

func init() {
	// 初始化全局日志配置
	zerolog.TimeFieldFormat = time.DateTime
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	DefaultLogger = New()
	consoleWriter = getConsoleWriter()
}

// SetGlobalLevel 设置全局日志级别
func SetGlobalLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// consoleWriter 创建控制台输出writer
func getConsoleWriter() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	output.FormatLevel = func(i any) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	return output
}

// New 创建新的Logger实例，输出到控制台
func New() *Logger {
	return &Logger{
		Logger: zerolog.New(consoleWriter).With().Timestamp().Caller().Logger(),
	}
}

// NewFile 创建文件输出的Logger
func NewFile(c Config) *Logger {
	if err := c.initConfig(); err != nil {
		// 如果配置初始化失败，返回一个只输出到控制台的Logger
		logger := New()
		logger.Error().Err(err).Msg("failed to initialize logger config")
		return logger
	}

	writer, err := rotateWriter(c)
	if err != nil {
		// 如果创建轮转writer失败，返回一个只输出到控制台的Logger
		logger := New()
		logger.Error().Err(err).Msg("failed to create rotate writer, using console output instead")
		return logger
	}

	return &Logger{
		Logger: zerolog.New(writer).With().Timestamp().Caller().Logger(),
	}
}

// NewMulti 创建同时输出到文件和控制台的Logger
func NewMulti(c Config) *Logger {
	if err := c.initConfig(); err != nil {
		// 如果配置初始化失败，返回一个只输出到控制台的Logger
		logger := New()
		logger.Error().Err(err).Msg("failed to initialize logger config")
		return logger
	}

	writer, err := rotateWriter(c)
	if err != nil {
		// 如果创建轮转writer失败，返回一个只输出到控制台的Logger
		logger := New()
		logger.Error().Err(err).Msg("failed to create rotate writer, using console output instead")
		return logger
	}

	multi := zerolog.MultiLevelWriter(writer, consoleWriter)

	return &Logger{
		Logger: zerolog.New(multi).With().Timestamp().Caller().Logger(),
	}
}

// fileFullPath 返回日志文件的完整路径
func (c *Config) fileFullPath() string {
	return c.fileFullPathWithFormat("")
}

// fileFullPathWithFormat 返回带格式的日志文件完整路径
func (c *Config) fileFullPathWithFormat(format string) string {
	var builder strings.Builder
	builder.Grow(len(c.Filename) + len(format) + len(c.FileExt) + 3)

	builder.WriteString(c.Filename)
	if format != "" {
		builder.WriteByte('.')
		builder.WriteString(format)
	}
	builder.WriteByte('.')
	builder.WriteString(c.FileExt)

	return filepath.Join(c.Filepath, builder.String())
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c)
}

// rotateWriter 日志轮转writer
func rotateWriter(config Config) (io.Writer, error) {
	switch config.RotateMode {
	case RotateModeTime:
		return timeRotateWriter(config)
	case RotateModeSize:
		return sizeRotateWriter(config)
	default:
		return nil, fmt.Errorf("unsupported rotate mode: %d", config.RotateMode)
	}
}

// timeRotateWriter 按时间轮转的writer
func timeRotateWriter(config Config) (io.Writer, error) {
	writer, err := rotatelogs.New(
		config.fileFullPathWithFormat("%Y%m%d%H%M"),
		rotatelogs.WithLinkName(config.fileFullPath()),
		rotatelogs.WithMaxAge(time.Duration(config.RotatelogsConfig.MaxAge)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(config.RotatelogsConfig.RotationTime)*time.Hour),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create time rotate writer: %w", err)
	}
	return writer, nil
}

// sizeRotateWriter 按大小轮转的writer
func sizeRotateWriter(config Config) (io.Writer, error) {

	return &lumberjack.Logger{
		Filename:   config.fileFullPath(),
		MaxSize:    config.LumberjackConfig.MaxSize,
		MaxBackups: config.LumberjackConfig.MaxBackups,
		MaxAge:     config.LumberjackConfig.MaxAge,
		Compress:   config.LumberjackConfig.Compress,
	}, nil
}
