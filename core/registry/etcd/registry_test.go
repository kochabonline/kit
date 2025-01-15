package etcd

import (
	"context"
	"testing"
	"time"

	"github.com/kochabonline/kit/core/registry"
	"github.com/kochabonline/kit/core/util"
	"github.com/kochabonline/kit/store/etcd"
)

func TestRegistry(t *testing.T) {
	e, err := etcd.New(&etcd.Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := NewRegistry(e.Client)
	wc, err := r.Watch(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			instance := <-wc
			for _, i := range instance {
				t.Log(i)
			}
		}
	}()
	r.Register(context.Background(), registry.Instance{Id: util.Id(), Name: "test", Endpoints: []string{"http://localhost:8080"}})
	time.Sleep(3 * time.Second)
	instance := registry.Instance{Id: util.Id(), Name: "test2", Endpoints: []string{"http://localhost:8080"}}
	r.Register(context.Background(), instance)
	time.Sleep(3 * time.Second)
	_ = r.Deregister(context.Background(), instance)
	time.Sleep(10 * time.Second)
}
