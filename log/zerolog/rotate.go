package zerolog

import (
	"io"
	"path"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"gopkg.in/natefinch/lumberjack.v2"
)

func rotate(config Config) io.Writer {
	var writer io.Writer
	var err error

	file := path.Join(config.Filepath, config.Filename)

	switch config.Mode {
	case ModeTime:
		writer, err = rotatelogs.New(
			file+".%Y%m%d%H%M",
			rotatelogs.WithLinkName(file),
			rotatelogs.WithMaxAge(time.Duration(config.MaxAgeDay)*24*time.Hour),
			rotatelogs.WithRotationTime(time.Duration(config.RotationTime)*time.Hour),
		)

		if err != nil {
			return nil
		}
	case ModeSize:
		writer = &lumberjack.Logger{
			Filename:   file,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}
	default:
		return nil
	}

	return writer
}
