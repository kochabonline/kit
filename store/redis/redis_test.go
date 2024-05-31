package redis

import "testing"

func TestSingle(t *testing.T) {
	r, err := NewClient(&Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCluster(t *testing.T) {
	r, err := NewClusterClient(&ClusterConfig{
		Addrs: []string{":6379", ":6380", ":6381"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}
