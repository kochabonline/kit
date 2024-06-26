package ioc

import (
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/kochabonline/kit/log"
)

type Store struct {
	namespaces map[string]namespace
	sorted     []*namespace
	log        *log.Helper
}

type namespace struct {
	name     string
	object   map[string]object
	sorted   []*object
	priority int
}

type object struct {
	ioc      Ioc
	priority int
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

func (s *Store) findNsMinPriority() int {
	minPriority := int(^uint(0) >> 1)

	for _, ns := range s.namespaces {
		if ns.priority < minPriority {
			minPriority = ns.priority
		}
	}

	return minPriority
}

func (s *Store) SetLogger(log *log.Helper) {
	s.log = log
}

func (s *Store) Init() error {
	s.sort()

	for _, ns := range s.sorted {
		ns.sort()
		objects := make([]string, 0, len(ns.object))
		for _, obj := range ns.sorted {
			if err := obj.ioc.Init(); err != nil {
				return err
			}
			objects = append(objects, obj.ioc.Name())
		}
		if len(objects) > 0 {
			s.log.Infof("[ioc] | namespace: %s | objects: %v", ns.name, objects)
		}
	}

	return nil
}

func (s *Store) RegisterNamespace(name string, priority int) {
	if priority <= s.findNsMinPriority() {
		priority = s.findNsMinPriority() + 1
	}

	s.namespaces[name] = namespace{name: name, object: map[string]object{}, priority: priority}
	s.log.Infof("[register] | namespace: %s | priority: %d", name, priority)
}

func (s *Store) Register(nsname string, ioc Ioc, priority ...int) {
	if _, ok := s.namespaces[nsname]; !ok {
		s.log.Errorf("namespace %s not found", nsname)
		return
	}

	p := 0
	if len(priority) > 0 {
		p = priority[0]
	}

	s.namespaces[nsname].object[ioc.Name()] = object{ioc: ioc, priority: p}
}

func (s *Store) Get(nsname string, name string) Ioc {
	if ns, ok := s.namespaces[nsname]; ok {
		if obj, ok := ns.object[name]; ok {
			return obj.ioc
		}
	}

	return nil
}

func (s *Store) GinIRouterRegister(r gin.IRouter) {
	for _, ns := range s.namespaces {
		routers := make([]string, 0, len(ns.object))
		for _, obj := range ns.object {
			if router, ok := obj.ioc.(GinIRouter); ok {
				router.Register(r)
			}
			routers = append(routers, obj.ioc.Name())
		}
		if len(routers) > 0 {
			s.log.Infof("[gin] | namespace: %s | routers: %v", ns.name, routers)
		}
	}
}
