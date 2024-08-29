package registry

type Registry interface {
	Register(name string, service any) error
	Deregister(name string) error
}
