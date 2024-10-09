package casbin

import (
	"testing"

	"github.com/kochabonline/kit/store/mysql"
)

func TestCasbin(t *testing.T) {
	m, err := mysql.New(&mysql.Config{
		Password: "12345678",
		DataBase: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err := New(Config{
		DB:    m.Client,
		Model: "./rbac_model.conf",
	})
	if err != nil {
		t.Fatal(err)
	}

	c.AddPolicies([]Rule{
		{Role: "alice2", Path: "/admin", Method: "GET"},
		{Role: "alice2", Path: "/admin", Method: "POST"},
	})
	c.SyncedCachedEnforcer.AddGroupingPolicy("admin", "alice2")
	c.SyncedCachedEnforcer.AddGroupingPolicy("admin", "alice")

	t.Log(c.GetGroupingPolicies("admin"))
	c.SyncedCachedEnforcer.AddRoleForUser("admin", "alice")
	c.SyncedCachedEnforcer.AddPermissionForUser("alice", "/admin", "GET")
	t.Log(c.Enforce("admin", "/admin", "GET"))
	c.UpdatePolicies([]Rule{
		{Role: "alice2", Path: "/admin", Method: "PUT"},
		{Role: "alice", Path: "/admin", Method: "POST"},
	}, []Rule{
		{Role: "alice2", Path: "/admin", Method: "POST"},
		{Role: "alice", Path: "/admin", Method: "DELETE"},
	})
}
