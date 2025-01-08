package captcha

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/kochabonline/kit/errors"
)

type Email struct {
	To      string
	Subject string
	Body    string
}

type SmtpPlainAuth struct {
	Identity string
	Username string
	Password string
	Host     string
	Port     int
}

type EmailAuthenticator struct {
	auth          smtp.Auth
	smtpPlainAuth SmtpPlainAuth
	cache         Cache
	prefix        string
	expires       time.Duration
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

func NewEmailAuthenticator(smtpPlainAuth SmtpPlainAuth, cache Cache, opts ...Option) *EmailAuthenticator {
	e := &EmailAuthenticator{
		auth:          smtp.PlainAuth(smtpPlainAuth.Identity, smtpPlainAuth.Username, smtpPlainAuth.Password, smtpPlainAuth.Host),
		prefix:        "email",
		smtpPlainAuth: smtpPlainAuth,
		cache:         cache,
		expires:       time.Minute * 10,
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (a *SmtpPlainAuth) addr() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}

func (e *EmailAuthenticator) key(suffix string) string {
	return fmt.Sprintf("%s:%s", e.prefix, suffix)
}

func (e *EmailAuthenticator) Send(ctx context.Context, email Email, code string) (time.Duration, error) {
	key := e.key(email.To)

	ttl, err := e.cache.GetTTL(ctx, key)
	if err != nil {
		return 0, errors.BadRequest("email captcha cache get ttl failed: %v", err)
	}
	if ttl > 0 {
		return ttl, nil
	}

	if err := e.cache.Set(ctx, key, code, e.expires); err != nil {
		return 0, errors.BadRequest("email captcha cache set failed: %v", err)
	}

	err = smtp.SendMail(
		e.smtpPlainAuth.addr(),
		e.auth, e.smtpPlainAuth.Username,
		[]string{email.To},
		[]byte(buildEmailMessage(email)),
	)
	if err != nil {
		if err := e.cache.Delete(ctx, key); err != nil {
			return 0, errors.BadRequest("email captcha cache delete failed: %v", err)
		}
		return 0, errors.BadRequest("email captcha send failed: %v", err)
	}

	return 0, nil
}

func buildEmailMessage(email Email) string {
	var builder strings.Builder
	builder.WriteString("To: ")
	builder.WriteString(email.To)
	builder.WriteString("\r\n")
	builder.WriteString("Subject: ")
	builder.WriteString(email.Subject)
	builder.WriteString("\r\n\r\n")
	builder.WriteString(email.Body)
	return builder.String()
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
