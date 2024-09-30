package mysql

import (
	"testing"
)

func TestMysql(t *testing.T) {
	m, err := New(&Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(m)
}
