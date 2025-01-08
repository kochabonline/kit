package etcd

import (
	"context"
	"testing"
	"time"
)

func TestEtcd(t *testing.T) {
	// Create a new etcd instance
	e, err := New(&Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()

	// Put a key
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = e.Client.Put(ctx, "sample_key", "sample_value")
	cancel()
	if err != nil {
		t.Fatal(err)
	}

	// Get the key
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := e.Client.Get(ctx, "sample_key")
	cancel()
	if err != nil {
		t.Fatal(err)
	}

	// Check if the key exists
	if len(resp.Kvs) == 0 {
		t.Fatal("not found")
	}
	if string(resp.Kvs[0].Value) != "sample_value" {
		t.Fatalf("expected value: sample_value, got: %s", string(resp.Kvs[0].Value))
	}

	t.Logf("key: %s, value: %s", string(resp.Kvs[0].Key), string(resp.Kvs[0].Value))
}
