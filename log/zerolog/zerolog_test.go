package zerolog

import (
	"testing"

	"github.com/kochabonline/kit/log/level"
)

func TestLogger(t *testing.T) {
	z := New()
	z.Log(level.Debug, "hello world")
	z.Log(level.Info, "hello world", "key", "value")
}
