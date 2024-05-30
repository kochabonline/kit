package ioc

import "testing"

type mock struct {
	Data string
}

func (m *mock) Init() error {
	return nil
}

func (m *mock) Name() string {
	return "mock"
}

type mock2 struct {
	Data string
}

func (m *mock2) Init() error {
	return nil
}

func (m *mock2) Name() string {
	return "mock2"
}

func TestStore(t *testing.T) {
	Container.RegisterNamespace("test", 1)
	Container.Register(ConfigNamespace, &mock{}, 10)
	Container.Register(DataBaseNamespace, &mock{}, 8)
	Container.Register(DataBaseNamespace, &mock2{}, 7)
	Container.Register(HandlerNamespace, &mock{}, 6)
	Container.Register(ControllerNamespace, &mock{}, 4)
	Container.Init()
}
