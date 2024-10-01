package tools

import (
	"context"
	"errors"
)

// CtxValue returns the value of the key from the context.
// It returns an error if the context is nil, the value is not found, or the value type is mismatched.
func CtxValue[T any](ctx context.Context, key any) (T, error) {
	var value T
	if ctx == nil {
		return value, errors.New("context is nil")
	}

	val := ctx.Value(key)
	if val == nil {
		return value, errors.New("value not found")
	}

	if value, ok := val.(T); ok {
		return value, nil
	}

	return value, errors.New("value type mismatch")
}
