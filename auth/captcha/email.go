package captcha

import (
	"context"
	"fmt"
	"time"

	"github.com/kochabonline/kit/core/bot/email"
	"github.com/kochabonline/kit/errors"
)

type Email struct {
	To      string
	Subject string
	Tip     string
	Code    string
}

type EmailAuthenticator struct {
	email   *email.Email
	cache   Cache
	prefix  string
	expires time.Duration
}

type Option func(*EmailAuthenticator)

func WithPrefix(prefix string) Option {
	return func(e *EmailAuthenticator) {
		e.prefix = prefix
	}
}

func WithExpires(expires time.Duration) Option {
	return func(e *EmailAuthenticator) {
		e.expires = expires
	}
}

func NewEmailAuthenticator(email *email.Email, cache Cache, opts ...Option) *EmailAuthenticator {
	e := &EmailAuthenticator{
		email:   email,
		prefix:  "email",
		cache:   cache,
		expires: time.Minute * 10,
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *EmailAuthenticator) key(suffix string) string {
	return fmt.Sprintf("%s:%s", e.prefix, suffix)
}

func (e *EmailAuthenticator) Send(ctx context.Context, em Email) (time.Duration, error) {
	key := e.key(em.To)

	ttl, err := e.cache.GetTTL(ctx, key)
	if err != nil {
		return 0, errors.BadRequest("email captcha cache get ttl failed: %v", err)
	}
	if ttl > 0 {
		return ttl, nil
	}

	if err := e.cache.Set(ctx, key, em.Code, e.expires); err != nil {
		return 0, errors.BadRequest("email captcha cache set failed: %v", err)
	}

	_, err = e.email.Send(email.NewMessage().With().
		To([]string{em.To}).
		Subject(em.Subject).
		Body(fmt.Sprintf("%s%s", em.Tip, em.Code)).
		Message())
	if err != nil {
		if err := e.cache.Delete(ctx, key); err != nil {
			return 0, errors.BadRequest("email captcha cache delete failed: %v", err)
		}
		return 0, errors.BadRequest("email captcha send failed: %v", err)
	}

	return 0, nil
}

func (e *EmailAuthenticator) Validate(ctx context.Context, email string, code string) (bool, error) {
	key := e.key(email)

	v, err := e.cache.Get(ctx, key)
	if err != nil {
		return false, errors.BadRequest("email captcha cache get failed: %v", err)
	}

	if v == code {
		if err := e.cache.Delete(ctx, key); err != nil {
			return false, errors.BadRequest("email captcha cache delete failed: %v", err)
		}
		return true, nil
	}

	return false, nil
}
