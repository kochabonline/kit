package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/kochabonline/kit/store/redis"
)

type Account struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

func TestSingle(t *testing.T) {
	r, err := redis.NewClient(&redis.Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	jwt, err := New(&Config{
		Expire:        6,
		RefreshExpire: 12,
	})
	if err != nil {
		t.Fatal(err)
	}

	jwtRedis := NewJwtRedis(jwt, r.Client, WithMultipleLogin(true))

	account := &Account{
		Id:       123,
		Username: "test",
	}
	claims := map[string]any{
		"username": account.Username,
		"id":   account.Id,
	}

	ctx := context.Background()

	auth, err := jwtRedis.Generate(ctx, claims)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Access token:", auth.AccessToken)
	t.Log("Refresh token:", auth.RefreshToken)

	time.Sleep(1 * time.Second)
	_, err = jwtRedis.Parse(ctx, auth.AccessToken)
	if err != nil {
		t.Fatal(err)
	}

	newAccessToken, err := jwtRedis.Refresh(ctx, auth.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("New access token:", newAccessToken)

	_, err = jwtRedis.Parse(ctx, newAccessToken)
	if err != nil {
		t.Fatal(err)
	}
	// jwtRedis.Delete(ctx, account.Id)
}
