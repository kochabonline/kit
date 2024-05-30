package mysql

import "testing"

func TestMysql(t *testing.T) {
	m, err := New(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(m)
}