package jwt

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/kochabonline/kit/store/redis"
)

const (
	accessTokenExpire  = 3
	refreshTokenExpire = 9
)

type User struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func TestStore(t *testing.T) {
	r, err := redis.NewClient(&redis.Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	jwt, err := New(&Config{
		Expire:        accessTokenExpire,
		RefreshExpire: refreshTokenExpire,
	})
	if err != nil {
		t.Fatal(err)
	}

	store := NewJwtRedisStore(r.Client)

	User := &User{
		Id:       1,
		Username: "test",
	}
	claims := map[string]any{
		"username": User.Username,
		"id":       User.Id,
	}
	accessToken, err := jwt.Generate(claims)
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := jwt.GenerateRefreshToken(claims)
	if err != nil {
		t.Fatal(err)
	}
	token := &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	tokenString, _ := json.Marshal(token)

	ctx := context.Background()
	err = store.Set(ctx, strconv.FormatInt(User.Id, 10), tokenString, refreshTokenExpire)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	getToken, err := store.Get(ctx, strconv.FormatInt(User.Id, 10))
	if err != nil {
		t.Fatal(err)
	}

	json.Unmarshal([]byte(getToken), &token)

	t.Log("token:", token)
}
