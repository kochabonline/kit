package ioc

import (
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
)

type Store struct {
	namespaces map[string]namespace
	sorted     []*namespace
}

type StoreOption func(*Store)

func WithNamespaces(namespaces map[string]namespace) StoreOption {
	return func(s *Store) {
		s.namespaces = namespaces
	}
}

type namespace struct {
	name     string
	object   map[string]object
	sorted   []*object
	priority int
}

type object struct {
	di       DependencyInjection
	priority int
}

func NewStore(opts ...StoreOption) *Store {
	store := &Store{
		namespaces: make(map[string]namespace),
	}

	for _, opt := range opts {
		opt(store)
	}

	return store
}

func (s *Store) sort() {
	s.sorted = make([]*namespace, 0, len(s.namespaces))

	for _, ns := range s.namespaces {
		s.sorted = append(s.sorted, &ns)
	}

	sort.Slice(s.sorted, func(i, j int) bool {
		return s.sorted[i].priority < s.sorted[j].priority
	})
}

func (n *namespace) sort() {
	n.sorted = make([]*object, 0, len(n.object))

	for _, obj := range n.object {
		n.sorted = append(n.sorted, &obj)
	}

	sort.Slice(n.sorted, func(i, j int) bool {
		return n.sorted[i].priority < n.sorted[j].priority
	})
}

func (s *Store) min() int {
	min := int(^uint(0) >> 1)

	for _, ns := range s.namespaces {
		if ns.priority < min {
			min = ns.priority
		}
	}

	return min
}

func (s *Store) Init() error {
	s.sort()

	for _, ns := range s.sorted {
		ns.sort()
		objects := make([]string, 0, len(ns.object))
		for _, obj := range ns.sorted {
			if err := obj.di.Init(); err != nil {
				return err
			}
			objects = append(objects, obj.di.Name())
		}
		if len(objects) > 0 {
			log.Infof("[ioc] | namespace: %s | objects: %v", ns.name, objects)
		}
	}

	return nil
}

func (s *Store) RegisterNamespace(name string) {
	priority := len(s.namespaces)
	s.namespaces[name] = namespace{name: name, object: map[string]object{}, priority: priority}
	log.Infof("[register] | namespace: %s | priority: %d", name, priority)
}

func (s *Store) RegisterNamespaceWithPriority(name string, priority int) {
	if priority <= s.min() {
		priority = s.min() + 1
	}

	s.namespaces[name] = namespace{name: name, object: map[string]object{}, priority: priority}
	log.Infof("[register] | namespace: %s | priority: %d", name, priority)
}

func (s *Store) Register(nsname string, di DependencyInjection) {
	if _, ok := s.namespaces[nsname]; !ok {
		log.Fatalf("register %s failed: namespace %s not found", di.Name(), nsname)
		return
	}

	priority := len(s.namespaces[nsname].object)
	s.namespaces[nsname].object[di.Name()] = object{di: di, priority: priority}
}

func (s *Store) RegisterWithPriority(nsname string, di DependencyInjection, priority int) {
	if _, ok := s.namespaces[nsname]; !ok {
		log.Fatalf("register %s failed: namespace %s not found", di.Name(), nsname)
		return
	}

	s.namespaces[nsname].object[di.Name()] = object{di: di, priority: priority}
}

func (s *Store) Get(nsname string, name string) DependencyInjection {
	if ns, ok := s.namespaces[nsname]; ok {
		if obj, ok := ns.object[name]; ok {
			return obj.di
		}
	}

	return nil
}

func (s *Store) GinIRouterRegister(r gin.IRouter) {
	for _, ns := range s.namespaces {
		routers := make([]string, 0, len(ns.object))
		for _, obj := range ns.object {
			if router, ok := obj.di.(GinIRouter); ok {
				router.Register(r)
			}
			routers = append(routers, obj.di.Name())
		}
		if len(routers) > 0 {
			log.Infof("[gin] | namespace: %s | routers: %v", ns.name, routers)
		}
	}
}
