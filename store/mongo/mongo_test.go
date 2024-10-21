package mongo

import "testing"

func TestMongo(t *testing.T) {
	m, err := New(&Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(m)
}
