package registry

import (
	"context"
	"encoding/json"
)

type Registerer interface {
	// Register registers the given instance with the registry.
	Register(ctx context.Context, instance Instance) error
	// Deregister deregisters the given instance from the registry.
	Deregister(ctx context.Context, instance Instance) error
}

type Discoverer interface {
	// Discover returns all instances of the service with the given name in the given namespace.
	Discover(ctx context.Context, name string) ([]*Instance, error)
	// Watch returns a channel that will receive all instances of the service with the given name in the given namespace.
	Watch(ctx context.Context, name string) (<-chan []*Instance, error)
}

type Instance struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	Endpoints []string          `json:"endpoints"`
	Metadata  map[string]string `json:"metadata"`
}

func (i *Instance) Marshal() (string, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (i *Instance) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), &i)
}
