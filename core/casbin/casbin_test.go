package casbin

import (
	"testing"

	"github.com/kochabonline/kit/store/mysql"
)

func TestCasbin(t *testing.T) {
	m, err := mysql.New(&mysql.Config{})
	if err != nil {
		t.Fatal(err)
	}

	c, err := New(Config{
		Db:    m.Client,
		Model: "./rbac_model.conf",
	})
	if err != nil {
		t.Fatal(err)
	}

	ok, err := c.E.AddPolicy("alice", "data1", "read")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ok)
	ok2, err := c.E.Enforce("alice", "data1", "read")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ok2)
}
