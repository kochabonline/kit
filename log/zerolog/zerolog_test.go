package zerolog

import (
	"testing"

	"github.com/kochabonline/kit/log/level"
)

func TestLogger(t *testing.T) {
	z := New(WithCaller())
	z.Log(level.Debug, "hello world")
}
