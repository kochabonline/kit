package queue

import "sync"

type SimpleQueue[T comparable] interface {
	Touch(item T)
	Push(item T)
	Len() int
	Pop() (item T)
}

type simpleQueue[T comparable] []T

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
