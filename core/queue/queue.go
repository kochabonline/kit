package queue

import "sync"

type SimpleQueue[T comparable] interface {
	// Touch is a no-op function that can be used to touch the queue
	Touch(item T)
	// Push adds an item to the queue
	Push(item T)
	// Len returns the number of items in the queue
	Len() int
	// Pop removes and returns the first item in the queue
	Pop() (item T)
}

// simpleQueue is a simple implementation of a queue
type simpleQueue[T comparable] []T

// DefaultQueue returns a new instance of a simpleQueue
func DefaultQueue[T comparable]() SimpleQueue[T] {
	return new(simpleQueue[T])
}

func (q *simpleQueue[T]) Touch(item T) {}

func (q *simpleQueue[T]) Push(item T) {
	*q = append(*q, item)
}

func (q *simpleQueue[T]) Len() int {
	return len(*q)
}

func (q *simpleQueue[T]) Pop() (item T) {
	item = (*q)[0]
	(*q)[0] = *new(T)
	*q = (*q)[1:]
	return item
}

type QueueInterface[T comparable] interface {
	Add(item T)
	Len() int
	Get() (item T, showdown bool)
	Done(item T)
	Showdown()
}

type Queue[T comparable] struct {
	queue simpleQueue[T]

	cond         *sync.Cond
	shuttingDown bool
}

func (q *Queue[T]) Add(item T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	if q.shuttingDown {
		return
	}

	q.queue.Push(item)
	q.cond.Signal()
}

func (q *Queue[T]) Len() int {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.queue.Len()
}

func (q *Queue[T]) Get() (item T, showdown bool) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for q.queue.Len() == 0 && !q.shuttingDown {
		q.cond.Wait()
	}

	if q.queue.Len() == 0 {
		return *new(T), true
	}

	item = q.queue.Pop()

	return item, false
}

func (q *Queue[T]) Done(item T) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.queue.Push(item)
	q.cond.Signal()
}

func (q *Queue[T]) Showdown() {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	q.shuttingDown = true
	q.cond.Broadcast()
}
