package ioc

import (
	"context"
	"testing"
)

type mock struct {
	Data string
}

func (m *mock) Init(ctx context.Context) error {
	return nil
}

func (m *mock) Name() string {
	return "mock"
}

type mock2 struct {
	Data string
}

func (m *mock2) Init(ctx context.Context) error {
	return nil
}

func (m *mock2) Name() string {
	return "mock2"
}

func TestStore(t *testing.T) {
	// Test namespace registration
	if err := Container.RegisterNamespace("test"); err != nil {
		t.Fatalf("Failed to register namespace: %v", err)
	}

	// Test object registration
	if err := Container.Register("test", &mock{Data: "test"}); err != nil {
		t.Fatalf("Failed to register mock: %v", err)
	}

	if err := Container.Register(ConfigNamespace, &mock{}); err != nil {
		t.Fatalf("Failed to register config mock: %v", err)
	}

	if err := Container.Register(DataBaseNamespace, &mock{}); err != nil {
		t.Fatalf("Failed to register database mock: %v", err)
	}

	if err := Container.Register(DataBaseNamespace, &mock2{}); err != nil {
		t.Fatalf("Failed to register database mock2: %v", err)
	}

	if err := Container.Register(HandlerNamespace, &mock{}); err != nil {
		t.Fatalf("Failed to register handler mock: %v", err)
	}

	if err := Container.Register(ControllerNamespace, &mock{}); err != nil {
		t.Fatalf("Failed to register controller mock: %v", err)
	}

	if err := Container.RegisterWithPriority(ControllerNamespace, &mock2{}, -1); err != nil {
		t.Fatalf("Failed to register controller mock2 with priority: %v", err)
	}

	// Test initialization
	if err := Container.Init(); err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	// Test retrieval
	obj := Container.Get("test", "mock")
	if obj == nil {
		t.Fatal("Failed to get mock object")
	}

	// Test utility methods
	if !Container.HasNamespace("test") {
		t.Fatal("Expected namespace 'test' to exist")
	}

	if !Container.HasObject("test", "mock") {
		t.Fatal("Expected object 'mock' to exist in namespace 'test'")
	}

	count := Container.CountInNamespace("test")
	if count != 1 {
		t.Fatalf("Expected 1 object in namespace 'test', got %d", count)
	}

	total := Container.Count()
	t.Logf("Total objects in container: %d", total)

	namespaces := Container.ListNamespaces()
	t.Logf("Namespaces: %v", namespaces)

	allObjects := Container.GetAll("test")
	if len(allObjects) != 1 {
		t.Fatalf("Expected 1 object from GetAll, got %d", len(allObjects))
	}

	t.Log("All tests passed")
}
