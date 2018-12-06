package network

import (
	"context"
	"sync"
)

type Queue struct {
	ch   chan bool
	buf  []context.Context
	w    int
	r    int
	mtx  sync.RWMutex
	wait bool
	size int
}

func NewQueue(size int) *Queue {
	if size < 1 {
		panic("queue size must be greater than zero")
	}
	q := &Queue{
		ch:   make(chan bool),
		buf:  make([]context.Context, size),
		w:    0,
		r:    0,
		size: size,
	}
	return q
}

func (q *Queue) Push(ctx context.Context) bool {
	defer q.mtx.Unlock()
	q.mtx.Lock()
	w := q.w
	if len(q.buf) > (w + 1) {
		w++
	} else {
		w = 0
	}
	if q.r == w {
		// log.Println("Queue.Push full")
		return false
	}
	q.buf[q.w] = ctx
	q.w = w
	if q.wait {
		q.wait = false
		q.ch <- true
	}
	return true
}

func (q *Queue) Pop() context.Context {
	defer q.mtx.Unlock()
	q.mtx.Lock()
	q.wait = false
	if q.w == q.r {
		// log.Println("Queue.Pop empty")
		return nil
	}
	ctx := q.buf[q.r].(context.Context)
	q.buf[q.r] = nil
	if len(q.buf) > (q.r + 1) {
		q.r++
	} else {
		q.r = 0
	}

	return ctx
}

func (q *Queue) Wait() <-chan bool {
	defer q.mtx.RUnlock()
	q.mtx.RLock()
	q.wait = true
	// log.Println("Queue.Wait")
	return q.ch
}

func (q *Queue) Available() int {
	defer q.mtx.RUnlock()
	q.mtx.RLock()
	if q.w < q.r {
		return q.size - q.r + q.w
	}
	return q.w - q.r
}

func (q *Queue) Size() int {
	return q.size
}
