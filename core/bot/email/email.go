package email

import (
	"fmt"
	"net/http"
	"net/mail"
	"net/smtp"
	"regexp"
	"strings"

	"github.com/kochabonline/kit/core/bot"
	"github.com/kochabonline/kit/errors"
	"github.com/russross/blackfriday/v2"
)

var _ bot.Bot = (*Email)(nil)

type SmtpPlainAuth struct {
	Identity string
	Username string
	Password string
	Host     string
	Port     int
}

func (s *SmtpPlainAuth) addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type Email struct {
	smtpPlainAuth SmtpPlainAuth
	auth          smtp.Auth
	to            []string
}

type Option func(*Email)

func WithTo(to []string) Option {
	return func(e *Email) {
		e.to = to
	}
}

func New(smtpPlainAuth SmtpPlainAuth, opts ...Option) *Email {
	e := &Email{
		smtpPlainAuth: smtpPlainAuth,
		auth:          smtp.PlainAuth(smtpPlainAuth.Identity, smtpPlainAuth.Username, smtpPlainAuth.Password, smtpPlainAuth.Host),
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *Email) Send(msg bot.Sendable) (*http.Response, error) {
	message, ok := msg.(*Message)
	if !ok {
		return nil, errors.BadRequest("invalid message")
	}

	headers := make(map[string]string, 4)
	from := mail.Address{Name: message.From, Address: e.smtpPlainAuth.Username}
	headers["From"] = from.String()
	var to []string
	if len(e.to) > 0 {
		if err := validEmails(e.to); err != nil {
			return nil, err
		}
		to = e.to
	}
	if len(message.To) > 0 {
		if err := validEmails(message.To); err != nil {
			return nil, err
		}
		to = message.To
	}
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = message.Subject
	headers["Content-Type"] = e.getContentType(message.Type)

	var builder strings.Builder
	for k, v := range headers {
		builder.WriteString(k)
		builder.WriteString(": ")
		builder.WriteString(v)
		builder.WriteString("\r\n")
	}
	builder.WriteString("\r\n")
	builder.WriteString(message.Body)

	body := []byte(builder.String())
	if message.Type == Markdown {
		body = blackfriday.Run(body)
	}

	return nil, smtp.SendMail(e.smtpPlainAuth.addr(), e.auth, e.smtpPlainAuth.Username, to, body)
}

func (e *Email) getContentType(ct string) string {
	switch ct {
	case Text:
		return "text/plain; charset=UTF-8"
	case Markdown:
		return "text/html; charset=UTF-8"
	default:
		return "text/plain; charset=UTF-8"
	}
}

func validEmails(to []string) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	for _, email := range to {
		if !re.MatchString(email) {
			return errors.BadRequest("invalid email address: %s", email)
		}
	}
	return nil
}
