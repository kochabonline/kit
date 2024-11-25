package scheduler

import (
	"context"
	"errors"
	"fmt"

	"go.etcd.io/etcd/client/v3"
)

type Scheduler struct {
	client        *clientv3.Client
	namespace     string
	taskNamespace string
	lockNamespace string
}

type Option func(*Scheduler)

func WithNamespace(namespace string) Option {
	return func(s *Scheduler) {
		s.namespace = namespace
	}
}

func WithTaskNamespace(namespace string) Option {
	return func(s *Scheduler) {
		s.taskNamespace = namespace
	}
}

func WithLockNamespace(namespace string) Option {
	return func(s *Scheduler) {
		s.lockNamespace = namespace
	}
}

func NewScheduler(client *clientv3.Client, opts ...Option) *Scheduler {
	scheduler := &Scheduler{
		client:        client,
		namespace:     "/scheduler",
		taskNamespace: "/tasks",
		lockNamespace: "/locks",
	}

	for _, opt := range opts {
		opt(scheduler)
	}

	return scheduler
}

func (s *Scheduler) taskKey(task *Task) string {
	return fmt.Sprintf("%s%s/%s", s.namespace, s.taskNamespace, task.Id)
}

func (s *Scheduler) lockKey(task *Task) string {
	return fmt.Sprintf("%s%s/%s", s.namespace, s.lockNamespace, task.Id)
}

func (s *Scheduler) lock(ctx context.Context, task *Task, ttl int64) error {
	resp, err := s.client.Grant(ctx, ttl)
	if err != nil {
		return err
	}

	_, err = s.client.Put(ctx, s.lockKey(task), "", clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}

	return s.keepAlive(ctx, resp.ID)
}

func (s *Scheduler) unlock(ctx context.Context, task *Task) error {
	resp, err := s.client.Get(ctx, s.lockKey(task))
	if err != nil {
		return err
	}

	if len(resp.Kvs) == 0 {
		return errors.New("lock not found")
	}

	leaseId := clientv3.LeaseID(resp.Kvs[0].Lease)
	if leaseId != 0 {
		if _, err := s.client.Revoke(ctx, leaseId); err != nil {
			return err
		}
	}
	_, err = s.client.Delete(ctx, s.lockKey(task))
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) keepAlive(ctx context.Context, leaseID clientv3.LeaseID) error {
	leaseChan, err := s.client.KeepAlive(ctx, leaseID)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case _, ok := <-leaseChan:
				if !ok {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (s *Scheduler) AddTask(ctx context.Context, task *Task) error {
	resp, err := s.client.Get(ctx, s.taskKey(task))
	if err != nil {
		return err
	}
	if resp.Count > 0 {
		return errors.New("task already exists")
	}

	_, err = s.client.Put(ctx, s.taskKey(task), task.Marshal())
	return err
}

func (s *Scheduler) RemoveTask(ctx context.Context, task *Task) error {
	_, err := s.client.Delete(ctx, s.taskKey(task))
	return err
}

func (s *Scheduler) DiscoverTasks(ctx context.Context) ([]*Task, error) {
	resp, err := s.client.Get(ctx, s.namespace, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, kv := range resp.Kvs {
		task := &Task{}
		if err := task.Unmarshal(string(kv.Value)); err != nil {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *Scheduler) WatchTasks(ctx context.Context) (<-chan []*Task, error) {
	watchChan := s.client.Watch(ctx, s.namespace, clientv3.WithPrefix())
	taskChan := make(chan []*Task)

	go func() {
		for {
			select {
			case resp, ok := <-watchChan:
				if !ok {
					close(taskChan)
					return
				}

				var tasks []*Task
				for _, event := range resp.Events {
					task := &Task{}
					if err := task.Unmarshal(string(event.Kv.Value)); err != nil {
						continue
					}

					tasks = append(tasks, task)
				}

				taskChan <- tasks
			case <-ctx.Done():
				close(taskChan)
				return
			}
		}
	}()

	return taskChan, nil
}
