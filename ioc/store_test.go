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
	Container.RegisterNamespace("test")
	Container.Register("test", &mock{Data: "test"})
	Container.Register(ConfigNamespace, &mock{})
	Container.Register(DataBaseNamespace, &mock{})
	Container.Register(DataBaseNamespace, &mock2{})
	Container.Register(HandlerNamespace, &mock{})
	Container.Register(ControllerNamespace, &mock{})
	Container.RegisterWithPriority(ControllerNamespace, &mock2{}, -1)
	Container.Init()
	t.Log(Container.Get("test", "mock"))
}
