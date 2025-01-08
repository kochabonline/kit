package casbin

import (
	"sync"
	"testing"
	"time"

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
		DB:         m.Client,
		Model:      "./rbac_model.conf",
		ExpireTime: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	c.AddPolicies([]Policy{
		{Role: "admin", Path: "/info", Method: "GET"},
		{Role: "normal", Path: "/info/:id", Method: "GET"},
	})
	c.SyncedCachedEnforcer.AddRoleForUser("alice", "admin")
	t.Log(c.GetGroupingPolicies("alice"))
	t.Log(c.Enforce("alice", "/info/1", "GET"))
	go func() {
		c.SyncedCachedEnforcer.AddRoleForUser("alice", "normal")
	}()
	time.Sleep(5 * time.Second)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Log(c.GetGroupingPolicies("alice"))
		t.Log(c.Enforce("alice", "/info/1", "GET"))
	}()
	wg.Wait()
}
