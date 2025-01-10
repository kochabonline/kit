package captcha

import (
	"context"
	"testing"

	"github.com/kochabonline/kit/core/bot/email"
	"github.com/kochabonline/kit/core/tools"
	"github.com/kochabonline/kit/store/redis"
)

func TestEmail(t *testing.T) {
	ctx := context.Background()
	r, _ := redis.NewClient(&redis.Config{
		Password: "12345678",
	})
	cache := NewRedisCache(r.Client)
	e := email.New(email.SmtpPlainAuth{})
	email := NewEmailAuthenticator(e, cache)

	code := tools.GenerateRandomCode(6)
	em := Email{
		To:      "",
		Subject: "code",
		Tip:     "Your verification code is: ",
		Code:    code,
	}
	ttl, err := email.Send(ctx, em)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ttl:", ttl)

	ok, err := email.Validate(ctx, "xxx", "549899")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ok:", ok)
}
