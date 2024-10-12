package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/kochabonline/kit/store/redis"
	"golang.org/x/sync/errgroup"
)

type User struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func TestSingle(t *testing.T) {
	r, err := redis.NewClient(&redis.Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	jwt, err := New(&Config{
		Expire:        2,
		RefreshExpire: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	jwtRedis := NewJwtRedis(jwt, r.Client)

	User := &User{
		Id:       1,
		Username: "test",
	}
	claims := map[string]any{
		"username": User.Username,
		"id":       User.Id,
	}
	accessJti := "123"
	accessToken, err := jwt.Generate(claims, WithJti(accessJti))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Access token:", accessToken)
	refreshJti := "456"
	refreshToken, err := jwt.GenerateRefreshToken(claims, WithJti(refreshJti))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Refresh token:", refreshToken)

	ctx := context.Background()
	jwtRedis.SetAccessJti(ctx, User.Id, accessJti)
	jwtRedis.SetRefreshJti(ctx, User.Id, refreshJti)

	aClaims, err := jwt.Parse(accessToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := jwtRedis.Get(ctx, User.Id, aClaims); err != nil {
		t.Fatal("Access token expired")
	}

	rClaims, err := jwt.Parse(refreshToken)
	if err != nil {
		t.Fatal(err)
	}

	if err := jwtRedis.Get(ctx, User.Id, rClaims); err != nil {
		t.Fatal("Refresh token expired")
	}

	time.Sleep(3 * time.Second)
}

func TestMultiple(t *testing.T) {
	r, err := redis.NewClient(&redis.Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	jwt, err := New(&Config{
		Expire:        3,
		RefreshExpire: 9,
	})
	if err != nil {
		t.Fatal(err)
	}

	jwtRedis := NewJwtRedis(jwt, r.Client, WithMultipleLogin(true))

	User := &User{
		Id:       1,
		Username: "test",
	}
	claims := map[string]any{
		"username": User.Username,
		"id":       User.Id,
	}

	var eg errgroup.Group
	eg.Go(func() error {
		accessJti := "789"
		accessToken, err := jwt.Generate(claims, WithJti(accessJti))
		if err != nil {
			return err
		}
		t.Log("Access token:", accessToken)
		refreshJti := "012"
		refreshToken, err := jwt.GenerateRefreshToken(claims, WithJti(refreshJti))
		if err != nil {
			t.Fatal(err)
		}
		t.Log("Refresh token:", refreshToken)

		ctx := context.Background()
		jwtRedis.SetAccessJti(ctx, User.Id, accessJti)
		jwtRedis.SetRefreshJti(ctx, User.Id, refreshJti)

		aClaims, err := jwt.Parse(accessToken)
		if err != nil {
			return err
		}
		if err := jwtRedis.Get(ctx, User.Id, aClaims); err != nil {
			return err
		}

		rClaims, err := jwt.Parse(refreshToken)
		if err != nil {
			return err
		}

		if err := jwtRedis.Get(ctx, User.Id, rClaims); err != nil {
			return err
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}

	accessJti := "123"
	accessToken, err := jwt.Generate(claims, WithJti(accessJti))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Access token:", accessToken)
	refreshJti := "456"
	refreshToken, err := jwt.GenerateRefreshToken(claims, WithJti(refreshJti))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Refresh token:", refreshToken)

	ctx := context.Background()
	jwtRedis.SetAccessJti(ctx, User.Id, accessJti)
	jwtRedis.SetRefreshJti(ctx, User.Id, refreshJti)

	aClaims, err := jwt.Parse(accessToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := jwtRedis.Get(ctx, User.Id, aClaims); err != nil {
		t.Fatal("Access token expired")
	}

	rClaims, err := jwt.Parse(refreshToken)
	if err != nil {
		t.Fatal(err)
	}

	if err := jwtRedis.Get(ctx, User.Id, rClaims); err != nil {
		t.Fatal("Refresh token expired")
	}
	time.Sleep(10 * time.Second)
}
