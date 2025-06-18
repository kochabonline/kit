package ioc

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
)

// Store represents the IoC container with thread-safe operations.
// It manages multiple namespaces, each containing dependency injection objects
// with priority-based initialization order.
type Store struct {
	mu         sync.RWMutex
	namespaces map[string]*namespace
	sorted     []*namespace
	dirty      bool // flag to indicate if sorting is needed
}

// StoreOption represents a functional option for Store configuration
type StoreOption func(*Store)

// WithNamespaces sets the initial namespaces for the store
func WithNamespaces(namespaces map[string]*namespace) StoreOption {
	return func(s *Store) {
		s.namespaces = namespaces
		s.dirty = true
	}
}

// namespace represents a collection of objects with priority.
// Objects within a namespace are initialized in priority order.
type namespace struct {
	mu       sync.RWMutex
	name     string
	objects  map[string]*object
	sorted   []*object
	priority int
	dirty    bool
}

// object represents a dependency injection object with priority
type object struct {
	di       DependencyInjection
	priority int
}

// NewStore creates a new IoC container with the given options
func NewStore(opts ...StoreOption) *Store {
	store := &Store{
		namespaces: make(map[string]*namespace),
		dirty:      true,
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

// ensureSorted ensures the store is sorted if dirty.
// Must be called with write lock held.
func (s *Store) ensureSorted() {
	if s.dirty {
		s.sortNamespaces()
		s.dirty = false
	}
}

// sortNamespaces sorts namespaces by priority (thread-unsafe, call with lock held).
// Optimized to reuse slice capacity when possible.
func (s *Store) sortNamespaces() {
	// Reuse existing slice if capacity is sufficient
	if cap(s.sorted) >= len(s.namespaces) {
		s.sorted = s.sorted[:0]
	} else {
		s.sorted = make([]*namespace, 0, len(s.namespaces))
	}

	for _, ns := range s.namespaces {
		s.sorted = append(s.sorted, ns)
	}

	sort.Slice(s.sorted, func(i, j int) bool {
		return s.sorted[i].priority < s.sorted[j].priority
	})
}

// ensureSorted ensures the namespace is sorted if dirty.
// Must be called with write lock held.
func (n *namespace) ensureSorted() {
	if n.dirty {
		n.sortObjects()
		n.dirty = false
	}
}

// sortObjects sorts objects by priority (thread-unsafe, call with lock held).
// Optimized to reuse slice capacity when possible.
func (n *namespace) sortObjects() {
	// Reuse existing slice if capacity is sufficient
	if cap(n.sorted) >= len(n.objects) {
		n.sorted = n.sorted[:0]
	} else {
		n.sorted = make([]*object, 0, len(n.objects))
	}

	for _, obj := range n.objects {
		n.sorted = append(n.sorted, obj)
	}

	sort.Slice(n.sorted, func(i, j int) bool {
		return n.sorted[i].priority < n.sorted[j].priority
	})
}

// Init initializes all registered objects in priority order.
// This method is thread-safe and ensures all objects are initialized
// in the correct order based on namespace and object priorities.
func (s *Store) Init() error {
	return s.InitWithContext(context.Background())
}

// InitWithContext initializes all registered objects with a context.
// This allows for timeout control and cancellation during initialization.
func (s *Store) InitWithContext(ctx context.Context) error {
	// Create a context with reasonable timeout if none is set
	if deadline, ok := ctx.Deadline(); !ok || deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Get sorted namespaces under read lock
	s.mu.Lock()
	s.ensureSorted()
	sortedNamespaces := make([]*namespace, len(s.sorted))
	copy(sortedNamespaces, s.sorted)
	s.mu.Unlock()

	// Initialize objects in each namespace
	for _, ns := range sortedNamespaces {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.initNamespace(ctx, ns); err != nil {
				return err
			}
		}
	}

	return nil
}

// initNamespace initializes all objects in a single namespace.
func (s *Store) initNamespace(ctx context.Context, ns *namespace) error {
	// Get sorted objects under read lock
	ns.mu.Lock()
	ns.ensureSorted()
	sortedObjects := make([]*object, len(ns.sorted))
	copy(sortedObjects, ns.sorted)
	ns.mu.Unlock()

	if len(sortedObjects) == 0 {
		return nil
	}

	// Initialize objects and collect names for logging
	objectNames := make([]string, 0, len(sortedObjects))
	for _, obj := range sortedObjects {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Initialize using the modern interface
			if err := obj.di.Init(ctx); err != nil {
				return fmt.Errorf("failed to initialize object %s in namespace %s: %w",
					obj.di.Name(), ns.name, err)
			}
			objectNames = append(objectNames, obj.di.Name())
		}
	}

	log.Infof("[ioc] | namespace: %s | objects: %v", ns.name, objectNames)
	return nil
}

// RegisterNamespace registers a new namespace with auto-generated priority.
// The priority is set to the current number of namespaces.
func (s *Store) RegisterNamespace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.namespaces[name]; exists {
		return fmt.Errorf("namespace %s already exists", name)
	}

	priority := len(s.namespaces)
	s.registerNamespaceUnsafe(name, priority)
	log.Infof("[register] | namespace: %s | priority: %d", name, priority)
	return nil
}

// RegisterNamespaceWithPriority registers a new namespace with specified priority.
// If the priority conflicts, it will be adjusted to be lower than the minimum.
func (s *Store) RegisterNamespaceWithPriority(name string, priority int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.namespaces[name]; exists {
		return fmt.Errorf("namespace %s already exists", name)
	}

	// Adjust priority if it conflicts
	minPriority := s.getMinPriorityUnsafe()
	if priority >= minPriority {
		priority = minPriority - 1
	}

	s.registerNamespaceUnsafe(name, priority)
	log.Infof("[register] | namespace: %s | priority: %d", name, priority)
	return nil
}

// registerNamespaceUnsafe registers a namespace without locking.
// Must be called with write lock held.
func (s *Store) registerNamespaceUnsafe(name string, priority int) {
	s.namespaces[name] = &namespace{
		name:     name,
		objects:  make(map[string]*object),
		priority: priority,
		dirty:    true,
	}
	s.dirty = true
}

// getMinPriorityUnsafe returns minimum priority without locking.
// Must be called with lock held.
func (s *Store) getMinPriorityUnsafe() int {
	if len(s.namespaces) == 0 {
		return 0
	}

	min := int(^uint(0) >> 1) // max int
	for _, ns := range s.namespaces {
		if ns.priority < min {
			min = ns.priority
		}
	}
	return min
}

// Register registers an object in the specified namespace with auto-generated priority.
// The priority is set to the current number of objects in the namespace.
func (s *Store) Register(nsname string, di DependencyInjection) error {
	ns, exists := s.getNamespaceForRegistration(nsname)
	if !exists {
		return fmt.Errorf("namespace %s not found when registering %s", nsname, di.Name())
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	if _, exists := ns.objects[di.Name()]; exists {
		return fmt.Errorf("object %s already exists in namespace %s", di.Name(), nsname)
	}

	priority := len(ns.objects)
	ns.objects[di.Name()] = &object{di: di, priority: priority}
	ns.dirty = true

	return nil
}

// RegisterWithPriority registers an object in the specified namespace with specified priority.
func (s *Store) RegisterWithPriority(nsname string, di DependencyInjection, priority int) error {
	ns, exists := s.getNamespaceForRegistration(nsname)
	if !exists {
		return fmt.Errorf("namespace %s not found when registering %s", nsname, di.Name())
	}

	ns.mu.Lock()
	defer ns.mu.Unlock()

	if _, exists := ns.objects[di.Name()]; exists {
		return fmt.Errorf("object %s already exists in namespace %s", di.Name(), nsname)
	}

	ns.objects[di.Name()] = &object{di: di, priority: priority}
	ns.dirty = true

	return nil
}

// getNamespaceForRegistration safely retrieves a namespace for registration.
func (s *Store) getNamespaceForRegistration(nsname string) (*namespace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, exists := s.namespaces[nsname]
	return ns, exists
}

// Get retrieves an object from the specified namespace.
// Returns nil if the namespace or object doesn't exist.
func (s *Store) Get(nsname string, name string) DependencyInjection {
	s.mu.RLock()
	ns, exists := s.namespaces[nsname]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if obj, exists := ns.objects[name]; exists {
		return obj.di
	}

	return nil
}

// GetAll returns all objects from the specified namespace.
// Returns nil if the namespace doesn't exist.
func (s *Store) GetAll(nsname string) []DependencyInjection {
	s.mu.RLock()
	ns, exists := s.namespaces[nsname]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if len(ns.objects) == 0 {
		return nil
	}

	objects := make([]DependencyInjection, 0, len(ns.objects))
	for _, obj := range ns.objects {
		objects = append(objects, obj.di)
	}

	return objects
}

// GetAllSorted returns all objects from the specified namespace in priority order.
// Returns nil if the namespace doesn't exist.
func (s *Store) GetAllSorted(nsname string) []DependencyInjection {
	s.mu.RLock()
	ns, exists := s.namespaces[nsname]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	ns.mu.Lock()
	ns.ensureSorted()
	sortedObjects := make([]*object, len(ns.sorted))
	copy(sortedObjects, ns.sorted)
	ns.mu.Unlock()

	if len(sortedObjects) == 0 {
		return nil
	}

	objects := make([]DependencyInjection, 0, len(sortedObjects))
	for _, obj := range sortedObjects {
		objects = append(objects, obj.di)
	}

	return objects
}

// ListNamespaces returns all namespace names.
// The returned slice is a copy and safe to modify.
func (s *Store) ListNamespaces() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.namespaces) == 0 {
		return nil
	}

	namespaces := make([]string, 0, len(s.namespaces))
	for name := range s.namespaces {
		namespaces = append(namespaces, name)
	}

	return namespaces
}

// HasNamespace checks if a namespace exists.
func (s *Store) HasNamespace(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.namespaces[name]
	return exists
}

// HasObject checks if an object exists in the specified namespace.
func (s *Store) HasObject(nsname, objname string) bool {
	s.mu.RLock()
	ns, exists := s.namespaces[nsname]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	_, exists = ns.objects[objname]
	return exists
}

// Count returns the total number of objects across all namespaces.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, ns := range s.namespaces {
		ns.mu.RLock()
		total += len(ns.objects)
		ns.mu.RUnlock()
	}

	return total
}

// CountInNamespace returns the number of objects in the specified namespace.
// Returns 0 if the namespace doesn't exist.
func (s *Store) CountInNamespace(nsname string) int {
	s.mu.RLock()
	ns, exists := s.namespaces[nsname]
	s.mu.RUnlock()

	if !exists {
		return 0
	}

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return len(ns.objects)
}

// GinRouterRegister registers all GinRouter objects with the gin router.
// This method processes all namespaces and registers objects that implement
// the GinRouter interface with proper error handling.
func (s *Store) GinRouterRegister(r gin.IRouter) error {
	// Get all namespaces
	s.mu.RLock()
	namespaces := make([]*namespace, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		namespaces = append(namespaces, ns)
	}
	s.mu.RUnlock()

	// Process each namespace
	for _, ns := range namespaces {
		if err := s.registerRoutersInNamespace(ns, r); err != nil {
			return fmt.Errorf("failed to register routes in namespace %s: %w", ns.name, err)
		}
	}
	return nil
}

// registerRoutersInNamespace registers all GinRouter objects in a single namespace.
func (s *Store) registerRoutersInNamespace(ns *namespace, r gin.IRouter) error {
	// Get all objects from the namespace
	ns.mu.RLock()
	objects := make([]*object, 0, len(ns.objects))
	for _, obj := range ns.objects {
		objects = append(objects, obj)
	}
	ns.mu.RUnlock()

	if len(objects) == 0 {
		return nil
	}

	// Register routers and collect names for logging
	routers := make([]string, 0, len(objects))
	for _, obj := range objects {
		// Check if component implements GinRouter interface
		if router, ok := obj.di.(GinRouter); ok {
			if err := router.RegisterGinRoutes(r); err != nil {
				return fmt.Errorf("failed to register routes for %s: %w", obj.di.Name(), err)
			}
			routers = append(routers, obj.di.Name())
		}
	}

	if len(routers) > 0 {
		log.Infof("[gin] | namespace: %s | routers: %v", ns.name, routers)
	}
	return nil
}

// Shutdown gracefully shuts down all components that implement the Destroyer interface.
// This method processes all namespaces in reverse priority order.
func (s *Store) Shutdown(ctx context.Context) error {
	// Create a context with reasonable timeout if none is set
	if deadline, ok := ctx.Deadline(); !ok || deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	// Get sorted namespaces under read lock (reverse order for shutdown)
	s.mu.Lock()
	s.ensureSorted()
	sortedNamespaces := make([]*namespace, len(s.sorted))
	copy(sortedNamespaces, s.sorted)
	s.mu.Unlock()

	// Shutdown objects in reverse order
	for i := len(sortedNamespaces) - 1; i >= 0; i-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.shutdownNamespace(ctx, sortedNamespaces[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// shutdownNamespace shuts down all objects in a single namespace.
func (s *Store) shutdownNamespace(ctx context.Context, ns *namespace) error {
	// Get sorted objects under read lock (reverse order for shutdown)
	ns.mu.Lock()
	ns.ensureSorted()
	sortedObjects := make([]*object, len(ns.sorted))
	copy(sortedObjects, ns.sorted)
	ns.mu.Unlock()

	if len(sortedObjects) == 0 {
		return nil
	}

	// Shutdown objects in reverse order and collect names for logging
	destroyedObjects := make([]string, 0, len(sortedObjects))
	for i := len(sortedObjects) - 1; i >= 0; i-- {
		obj := sortedObjects[i]
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if destroyer, ok := obj.di.(Destroyer); ok {
				if err := destroyer.Destroy(ctx); err != nil {
					return fmt.Errorf("failed to destroy object %s in namespace %s: %w",
						obj.di.Name(), ns.name, err)
				}
				destroyedObjects = append(destroyedObjects, obj.di.Name())
			}
		}
	}

	if len(destroyedObjects) > 0 {
		log.Infof("[ioc] | shutdown namespace: %s | objects: %v", ns.name, destroyedObjects)
	}
	return nil
}

// HealthCheck performs health checks on all registered components.
// Returns the first error encountered, if any.
func (s *Store) HealthCheck(ctx context.Context) error {
	// Get all namespaces
	s.mu.RLock()
	namespaces := make([]*namespace, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		namespaces = append(namespaces, ns)
	}
	s.mu.RUnlock()

	// Check health of all components
	for _, ns := range namespaces {
		if err := s.healthCheckNamespace(ctx, ns); err != nil {
			return err
		}
	}

	return nil
}

// healthCheckNamespace performs health checks on all objects in a single namespace.
func (s *Store) healthCheckNamespace(ctx context.Context, ns *namespace) error {
	// Get all objects from the namespace
	ns.mu.RLock()
	objects := make([]*object, 0, len(ns.objects))
	for _, obj := range ns.objects {
		objects = append(objects, obj)
	}
	ns.mu.RUnlock()

	// Perform health checks
	for _, obj := range objects {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if checker, ok := obj.di.(HealthChecker); ok {
				if err := checker.HealthCheck(ctx); err != nil {
					return fmt.Errorf("health check failed for %s in namespace %s: %w",
						obj.di.Name(), ns.name, err)
				}
			}
		}
	}

	return nil
}
