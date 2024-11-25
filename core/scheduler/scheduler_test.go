package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kochabonline/kit/store/etcd"
	"github.com/kochabonline/kit/store/redis"
)

func TestGossip(t *testing.T) {
	tasks := []Task{
		{Id: "task1", Name: "task1", Status: "pending"},
		{Id: "task2", Name: "task2", Status: "pending"},
		{Id: "task3", Name: "task3", Status: "pending"},
		{Id: "task4", Name: "task4", Status: "pending"},
		{Id: "task5", Name: "task5", Status: "pending"},
	}
	r, err := redis.NewClient(&redis.Config{
		Password: "12345678",
	})
	if err != nil {
		t.Fatal(err)
	}
	lock := NewDistributedLock(r.Client)
	gossip := NewGossip()
	node := NewNode("node1", lock, tasks)
	node2 := NewNode("node2", lock, tasks)
	gossip.AddNode(node)
	gossip.AddNode(node2)

	go node.ScheduleTasks()
	go node2.ScheduleTasks()
	time.Sleep(1 * time.Second)
	gossip.RemoveNode(node)
	time.Sleep(10 * time.Second)
}

type T1 struct {
	commond string
}

func (t *T1) Execute() error {
	fmt.Println(t.commond)
	return nil
}

func TestScheduler(t *testing.T) {
	e, err := etcd.New(&etcd.Config{Password: "12345678"})
	if err != nil {
		t.Fatal(err)
	}
	scheduler := NewScheduler(e.Client)
	ctx := context.Background()
	taskChan, err := scheduler.WatchTasks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			tasks := <-taskChan

			for _, task := range tasks {
				t.Log("watched task:", task)
			}
		}
	}()

	// tasks, err := scheduler.DiscoverTasks(ctx)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// for _, task := range tasks {
	// 	t.Log("discovered task:", task)
	// }

	task1 := &Task{Id: "task1", Name: "task1", Status: "pending", Excutor: Excutor{Excute: "echo hello"}}
	if err = scheduler.AddTask(ctx, task1); err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)
}
