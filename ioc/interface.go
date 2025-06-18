package ioc

import (
	"context"

	"github.com/gin-gonic/gin"
)

// Component defines the basic contract for all IoC components.
// This interface follows the principle of small, focused interfaces.
type Component interface {
	// Name returns the unique identifier for this component.
	// It should be immutable and unique within its namespace.
	Name() string
}

// Initializer defines components that require initialization.
// This is separated from Component to follow the Interface Segregation Principle.
type Initializer interface {
	// Init initializes the component with context support for cancellation and timeout.
	// It should be idempotent and safe to call multiple times.
	Init(ctx context.Context) error
}

// Destroyer defines components that require cleanup during shutdown.
// This enables proper resource management and graceful shutdowns.
type Destroyer interface {
	// Destroy cleans up resources used by the component.
	// It should be idempotent and safe to call multiple times.
	Destroy(ctx context.Context) error
}

// HealthChecker defines components that can report their health status.
// This is useful for monitoring and health check endpoints.
type HealthChecker interface {
	// HealthCheck returns the current health status of the component.
	// It should be lightweight and non-blocking.
	HealthCheck(ctx context.Context) error
}

// DependencyInjection is a composite interface for components in the IoC container.
// It combines Component and Initializer interfaces for a complete dependency injection solution.
type DependencyInjection interface {
	Component
	Initializer
}

// HTTPRouter defines components that register HTTP routes.
// This interface is more generic and not tied to Gin specifically.
type HTTPRouter interface {
	Component
	// RegisterRoutes registers HTTP routes with the given router.
	RegisterRoutes(router any) error
}

// GinRouter defines Gin-specific router registration.
// This is the preferred interface for Gin route registration.
type GinRouter interface {
	Component
	// RegisterGinRoutes registers routes with a Gin router.
	RegisterGinRoutes(r gin.IRouter) error
}

// Priority defines components that have initialization/execution priority.
// Lower values indicate higher priority (executed first).
type Priority interface {
	// Priority returns the priority value for this component.
	// Components with lower priority values are processed first.
	Priority() int
}

// ConfigProvider defines components that provide configuration.
// This enables components to share configuration in a type-safe way.
type ConfigProvider interface {
	Component
	// GetConfig returns the configuration object.
	// The returned value should be immutable or a copy to prevent external modifications.
	GetConfig() any
}

// ServiceProvider defines components that provide services to other components.
// This enables loose coupling between components.
type ServiceProvider interface {
	Component
	// GetService returns a service by its type or name.
	// Returns nil if the service is not available.
	GetService(serviceType string) any
}
