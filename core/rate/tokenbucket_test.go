package rate

import (
	"fmt"
	"testing"
	"time"

	"github.com/kochabonline/kit/store/redis"
)

func TestAllow(t *testing.T) {
	r, err := redis.NewClient(&redis.Config{
		Password: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	bucket := NewTokenBucketLimiter(r.Client, "test", 2, 1)
	for i := 0; i < 1000; i++ {
		if bucket.Allow() {
			fmt.Printf("Request allowed [%v]\n", time.Now().Second())
		} else {
			fmt.Printf("Request denied [%v]\n", time.Now().Second())
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func TestAllowN(t *testing.T) {
	r, err := redis.NewClient(&redis.Config{
		Password: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	bucket := NewTokenBucketLimiter(r.Client, "test", 5, 1)
	for i := 0; i < 1000; i++ {
		if bucket.AllowN(time.Now(), 2) {
			fmt.Println("Request allowed")
		} else {
			fmt.Println("Request denied")
		}
		time.Sleep(500 * time.Millisecond)
	}
}
