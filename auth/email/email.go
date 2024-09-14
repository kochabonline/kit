package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/kochabonline/kit/errors"
	"golang.org/x/sync/errgroup"
)

const (
	ErrEmailSendFailedReason     = "email send failed"
	ErrEmailValidateFailedReason = "email validate failed"
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

func (e *EmailAuthenticator) Send(email Email, code string) (time.Duration, error) {
	key := e.key(email.To)

	ttl, err := e.cache.GetTTL(key)
	if err != nil {
		return 0, errors.BadRequest(ErrEmailSendFailedReason, "email captcha cache get ttl failed: %v", err)
	}
	if ttl > 0 {
		return ttl, nil
	}

	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		if err := e.cache.Set(key, code, e.expires); err != nil {
			return errors.BadRequest(ErrEmailSendFailedReason, "email captcha cache set failed: %v", err)
		}
		err := smtp.SendMail(
			e.smtpPlainAuth.addr(),
			e.auth, e.smtpPlainAuth.Username,
			[]string{email.To},
			[]byte(buildEmailMessage(email)),
		)
		if err != nil {
			if err := e.cache.Delete(key); err != nil {
				return errors.BadRequest(ErrEmailSendFailedReason, "email captcha cache delete failed: %v", err)
			}
			return errors.BadRequest(ErrEmailSendFailedReason, "%v", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return 0, err
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

func (e *EmailAuthenticator) Validate(email string, code string) (bool, error) {
	key := e.key(email)

	v, err := e.cache.Get(key)
	if err != nil {
		return false, errors.BadRequest(ErrEmailValidateFailedReason, "email captcha cache get failed: %v", err)
	}

	if v == code {
		if err := e.cache.Delete(key); err != nil {
			return false, errors.BadRequest(ErrEmailValidateFailedReason, "email captcha cache delete failed: %v", err)
		}
		return true, nil
	}

	return false, nil
}
