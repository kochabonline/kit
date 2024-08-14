package zerolog

import (
	"io"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"gopkg.in/natefinch/lumberjack.v2"
)

func rotate(config Config) io.Writer {
	var writer io.Writer
	var err error

	switch config.Mode {
	case ModeTime:
		writer, err = rotatelogs.New(
			config.fileFullPathWithDataTimeFormat("%Y%m%d%H%M"),
			rotatelogs.WithLinkName(config.fileFullPath()),
			rotatelogs.WithMaxAge(time.Duration(config.RotatelogsConfig.MaxAge)*time.Hour),
			rotatelogs.WithRotationTime(time.Duration(config.RotatelogsConfig.RotationTime)*time.Hour),
		)

		if err != nil {
			return nil
		}
	case ModeSize:
		writer = &lumberjack.Logger{
			Filename:   config.fileFullPath(),
			MaxSize:    config.LumberjackConfig.MaxSize,
			MaxBackups: config.LumberjackConfig.MaxBackups,
			MaxAge:     config.LumberjackConfig.MaxAge,
			Compress:   config.LumberjackConfig.Compress,
		}
	default:
		return nil
	}

	return writer
}
