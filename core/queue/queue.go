package queue

import (
	"sync"
)

type TypedInterface[T comparable] interface {
	Add(item T)
	Len() int
	Get() (item T, showdown bool)
	Done(item T)
	Showdown()
	ShutDownWithDrain()
	ShuttingDown() bool
}

type Queue[T comparable] interface {
	// Touch is a no-op function that can be used to touch the queue
	Touch(item T)
	// Push adds an item to the queue
	Push(item T)
	// Len returns the number of items in the queue
	Len() int
	// Pop removes and returns the first item in the queue
	Pop() (item T)
}

// DefaultQueue returns a new queue
func DefaultQueue[T comparable]() Queue[T] {
	return new(queue[T])
}

// simpleQueue is a simple implementation of a queue
type queue[T comparable] []T

func (q *queue[T]) Touch(item T) {}

func (q *queue[T]) Push(item T) {
	*q = append(*q, item)
}

func (q *queue[T]) Len() int {
	return len(*q)
}

func (q *queue[T]) Pop() (item T) {
	item = (*q)[0]
	(*q)[0] = *new(T)
	*q = (*q)[1:]
	return item
}

type TypedQueueConfig[T comparable] struct {
	// Name is the name of the queue
	Name string
	// Queue is the queue implementation
	Queue Queue[T]
}

func NewTyped[T comparable]() *Typed[T] {
	return NewTypedWithConfig(TypedQueueConfig[T]{
		Name: "",
	})
}

func NewTypedWithConfig[T comparable](config TypedQueueConfig[T]) *Typed[T] {
	return newQueueWithConfig(config)
}

func newQueueWithConfig[T comparable](config TypedQueueConfig[T]) *Typed[T] {
	if config.Queue == nil {
		config.Queue = DefaultQueue[T]()
	}

	return newQueue(
		config.Queue,
	)
}

func newQueue[T comparable](queue Queue[T]) *Typed[T] {
	return &Typed[T]{
		queue:      queue,
		dirty:      set[t]{},
		processing: set[t]{},
		cond:       sync.NewCond(&sync.Mutex{}),
	}
}

type Typed[T comparable] struct {
	queue      Queue[T]
	dirty      set[t]
	processing set[t]

	cond *sync.Cond

	shuttingDown bool
	drain        bool
}

type empty struct{}
type t any
type set[t comparable] map[t]empty

func (s set[t]) has(item t) bool {
	_, exists := s[item]
	return exists
}

func (s set[t]) insert(item t) {
	s[item] = empty{}
}

func (s set[t]) delete(item t) {
	delete(s, item)
}

func (s set[t]) len() int {
	return len(s)
}

func (q *Typed[T]) Add(item T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	if q.shuttingDown {
		return
	}

	if q.dirty.has(item) {
		if !q.processing.has(item) {
			q.queue.Touch(item)
		}
		return
	}

	q.dirty.insert(item)
	if q.processing.has(item) {
		return
	}

	q.queue.Push(item)
	q.cond.Signal()
}

func (q *Typed[T]) Len() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.queue.Len()
}

func (q *Typed[T]) Get() (item T, showdown bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for q.queue.Len() == 0 && !q.shuttingDown {
		q.cond.Wait()
	}

	if q.queue.Len() == 0 {
		return *new(T), true
	}

	item = q.queue.Pop()

	q.processing.insert(item)
	q.dirty.delete(item)

	return item, false
}

func (q *Typed[T]) Done(item T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.processing.delete(item)
	if q.dirty.has(item) {
		q.queue.Push(item)
		q.cond.Signal()
	} else if q.processing.len() == 0 {
		q.cond.Signal()
	}
}

func (q *Typed[T]) Showdown() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.drain = false
	q.shuttingDown = true
	q.cond.Broadcast()
}

func (q *Typed[T]) ShutDownWithDrain() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.drain = true
	q.shuttingDown = true
	q.cond.Broadcast()
}

func (q *Typed[T]) ShuttingDown() bool {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	return q.shuttingDown
}
