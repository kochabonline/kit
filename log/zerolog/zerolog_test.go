package zerolog

import (
	"testing"

	"github.com/kochabonline/kit/log/level"
)

func TestLogger(t *testing.T) {
	z := New()
	z.Log(level.Debug, "key", "value")
	z.Log(level.Info, "msg", "test")
}
