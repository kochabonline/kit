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

var (
	DefaultLogger *Logger
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
	desensitizeHook *DesensitizeHook
}

type Option func(*Logger)

// WithCaller 设置调用栈信息
func WithCaller() Option {
	return func(l *Logger) {
		l.Logger = l.Logger.With().Caller().Logger()
	}
}

// WithCallerSkip 设置调用栈跳过的帧数
func WithCallerSkip(skip int) Option {
	return func(l *Logger) {
		l.Logger = l.Logger.With().CallerWithSkipFrameCount(skip).Logger()
	}
}

// WithDesensitize 设置脱敏钩子
func WithDesensitize(hook *DesensitizeHook) Option {
	return func(l *Logger) {
		l.desensitizeHook = hook
	}
}

// GetDesensitizeHook 获取脱敏钩子
func (l *Logger) GetDesensitizeHook() *DesensitizeHook {
	return l.desensitizeHook
}

// SetGlobalLevel 设置全局日志级别
func SetGlobalLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

func init() {
	// 初始化全局日志配置
	zerolog.TimeFieldFormat = time.DateTime
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	DefaultLogger = New()
}

// newFallbackWriter 创建一个回退的日志writer
func newFallbackWriter(config Config) io.Writer {
	if err := config.initConfig(); err != nil {
		return consoleWriter()
	}

	writer, err := rotateWriter(config)
	if err != nil {
		return consoleWriter()
	}

	return writer
}

func newBaseLogger(writer io.Writer) *Logger {
	return &Logger{
		Logger: zerolog.New(writer).With().Timestamp().Logger(),
	}
}

func (l *Logger) handlerDesensitizeHook(writer io.Writer, opts ...Option) {
	if l.desensitizeHook != nil {
		w := NewDesensitizeWriter(writer, l.desensitizeHook)
		l.Logger = zerolog.New(w).With().Timestamp().Logger()
	}

	// 重建 logger 后，重新应用所有选项
	for _, opt := range opts {
		opt(l)
	}
}

// New 创建新的Logger实例，输出到控制台
func New(opts ...Option) *Logger {
	logger := newBaseLogger(consoleWriter())

	for _, opt := range opts {
		opt(logger)
	}

	logger.handlerDesensitizeHook(os.Stdout, opts...)

	return logger
}

// NewFile 创建文件输出的Logger
func NewFile(c Config, opts ...Option) *Logger {
	writer := newFallbackWriter(c)
	logger := newBaseLogger(writer)

	for _, opt := range opts {
		opt(logger)
	}

	logger.handlerDesensitizeHook(writer, opts...)

	return logger
}

// NewMulti 创建同时输出到文件和控制台的Logger
func NewMulti(c Config, opts ...Option) *Logger {
	writer := newFallbackWriter(c)
	multi := zerolog.MultiLevelWriter(writer, consoleWriter())
	logger := newBaseLogger(multi)

	for _, opt := range opts {
		opt(logger)
	}

	logger.handlerDesensitizeHook(multi, opts...)

	return logger
}

func (c *Config) initConfig() error {
	return reflect.SetDefaultTag(c)
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

// consoleWriter 创建控制台输出writer
func consoleWriter() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	output.FormatLevel = func(i any) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	return output
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
