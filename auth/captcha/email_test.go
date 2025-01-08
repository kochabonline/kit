package captcha

import (
	"context"
	"fmt"
	"testing"

	"github.com/kochabonline/kit/core/tools"
	"github.com/kochabonline/kit/store/redis"
)

func TestEmail(t *testing.T) {
	ctx := context.Background()
	r, _ := redis.NewClient(&redis.Config{
		Password: "xxx",
	})
	cache := NewRedisCache(r.Client)
	auth := SmtpPlainAuth{
		Password: "xxxx",
		Username: "xxx",
		Host:     "xxx",
		Port:     25,
	}
	email := NewEmailAuthenticator(auth, cache)

	code := tools.GenerateRandomCode(6)
	e := Email{
		To:      "xxx",
		Subject: "code",
		Body:    fmt.Sprintf("xxx：%s", code),
	}
	ttl, err := email.Send(ctx, e, code)
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
