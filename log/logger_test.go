package log

import (
	"encoding/json"
	"testing"

	"github.com/kochabonline/kit/errors"
)

type mock struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (m *mock) String() string {
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

func TestLog(t *testing.T) {
	logger := New()
	logger.Debug().Msg("test debug message")
	logger.Info().Str("key", "value").Msg("test info with field")
	logger.Error().Err(errors.New(400, "test")).Msg("test error")

	m := &mock{Name: "test", Age: 10}
	logger.Info().Any("user", m).Msg("test with struct")
}

func TestGlobalLog(t *testing.T) {
	Debug().Msg("test global debug log")
	Info().Msg("test global info log")
	Warn().Msg("test global warn log")
	Warn().Err(errors.New(404, "test warn error")).Msg("test global warn error log")
	Error().Err(errors.New(500, "test global error")).Msg("test global error log")
	Fatal().Msg("test global fatal log")
	Panic().Msg("test global panic log")
}

func TestFileLog(t *testing.T) {
	config := Config{
		RotateMode: RotateModeSize,
		Filename:   "test",
		FileExt:    "log",
		LumberjackConfig: LumberjackConfig{
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
	}

	logger := NewFile(config)
	logger.Info().Msg("test file log")
	logger.Info().Str("phone", "1234567890").Msg("test desensitize phone")
}

func TestMultiLog(t *testing.T) {
	config := Config{
		RotateMode: RotateModeTime,
		Filename:   "multi",
		FileExt:    "log",
		RotatelogsConfig: RotatelogsConfig{
			MaxAge:       24,
			RotationTime: 1,
		},
	}

	logger := NewMulti(config)
	logger.Info().Str("type", "multi").Msg("test multi output log")
}
