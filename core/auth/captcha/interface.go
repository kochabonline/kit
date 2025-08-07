package captcha

import (
	"context"
	"time"
)

type Cache interface {
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Get(ctx context.Context, key string) (any, error)
	GetTTL(ctx context.Context, key string) (time.Duration, error)
	Delete(ctx context.Context, key string) error
}

type EmailAuthenticatorer interface {
	Send(ctx context.Context, email Email, code string) error
	Validate(ctx context.Context, email string, code string) (bool, error)
}
