package etcd

import (
	"context"
	"fmt"

	"github.com/kochabonline/kit/core/registry"
	"go.etcd.io/etcd/client/v3"
)

var _ registry.Registerer = (*Registry)(nil)
var _ registry.Discoverer = (*Registry)(nil)

type Registry struct {
	client    *clientv3.Client
	ttl       int64
	namespace string
}

type Option func(*Registry)

func WithTTL(ttl int64) Option {
	return func(r *Registry) {
		r.ttl = ttl
	}
}

func WithNamespace(namespace string) Option {
	return func(r *Registry) {
		r.namespace = namespace
	}
}

func NewRegistry(client *clientv3.Client, opts ...Option) *Registry {
	r := &Registry{
		client:    client,
		ttl:       5,
		namespace: "/services",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Registry) Register(ctx context.Context, instance registry.Instance) error {
	resp, err := r.client.Grant(ctx, r.ttl)
	if err != nil {
		return err
	}

	value, err := instance.Marshal()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s/%s", r.namespace, instance.Name, instance.Id)
	_, err = r.client.Put(ctx, key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}

	return r.keepAlive(ctx, resp.ID)
}

func (r *Registry) Deregister(ctx context.Context, instance registry.Instance) error {
	key := fmt.Sprintf("%s/%s/%s", r.namespace, instance.Name, instance.Id)
	resp, err := r.client.Get(ctx, key)
	if err != nil {
		return err
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("instance not found: %v", instance)
	}

	leaseId := clientv3.LeaseID(resp.Kvs[0].Lease)
	if leaseId != 0 {
		if _, err := r.client.Revoke(ctx, leaseId); err != nil {
			return err
		}
	}
	_, err = r.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registry) keepAlive(ctx context.Context, id clientv3.LeaseID) error {
	leaseChan, err := r.client.KeepAlive(ctx, id)
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

func (r *Registry) Discover(ctx context.Context, name string) ([]*registry.Instance, error) {
	key := fmt.Sprintf("%s/%s", r.namespace, name)
	resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var instances []*registry.Instance
	for _, kv := range resp.Kvs {
		var instance registry.Instance
		if err := instance.Unmarshal(string(kv.Value)); err != nil {
			return nil, err
		}

		instances = append(instances, &instance)
	}

	return instances, nil
}

func (r *Registry) Watch(ctx context.Context, name string) (<-chan []*registry.Instance, error) {
	key := fmt.Sprintf("%s/%s", r.namespace, name)
	watchChan := r.client.Watch(ctx, key, clientv3.WithPrefix())

	instanceChan := make(chan []*registry.Instance)
	go func() {
		for {
			select {
			case resp, ok := <-watchChan:
				if !ok {
					close(instanceChan)
					return
				}

				var instances []*registry.Instance
				for _, event := range resp.Events {
					var instance registry.Instance
					if err := instance.Unmarshal(string(event.Kv.Value)); err != nil {
						continue
					}

					instances = append(instances, &instance)
				}

				instanceChan <- instances
			case <-ctx.Done():
				close(instanceChan)
				return
			}
		}
	}()

	return instanceChan, nil
}
