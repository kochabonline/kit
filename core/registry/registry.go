package registry

import (
	"context"
)

type Registerer interface {
	// Register registers the given instance with the registry.
	Register(ctx context.Context, instance Instance) error
	// Deregister deregisters the given instance from the registry.
	Deregister(ctx context.Context, instance Instance) error
}

type Discoverer interface {
	// Discover returns all instances of the service with the given name in the given namespace.
	Discover(ctx context.Context, namespace string, name string) ([]*Instance, error)
}

type Instance struct {
	Id        string
	Name      string
	Namespace string
	Endpoints []string
}
