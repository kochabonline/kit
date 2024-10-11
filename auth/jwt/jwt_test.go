package jwt

import (
	"testing"
	"time"
)

func TestJwt(t *testing.T) {
	jwt, err := New(&Config{
		Expire:        1,
		RefreshExpire: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	userClaims := map[string]any{
		"username": "admin",
	}
	tokenString, err := jwt.Generate(userClaims)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Token:", tokenString)
	refreshTokenString, err := jwt.GenerateRefreshToken(userClaims)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	newTokenString, err := jwt.Refresh(refreshTokenString)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("New token:", newTokenString)
}
