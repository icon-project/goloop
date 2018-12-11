package network

import (
	"context"
	"sync"
)

type Queue struct {
	buf     []context.Context
	w       int
	r       int
	len     int
	size    int
	mtx     sync.RWMutex
	wait    map[chan bool]interface{}
	mtxWait sync.Mutex
}

func NewQueue(size int) *Queue {
	if size < 1 {
		panic("queue size must be greater than zero")
	}
	q := &Queue{
		buf:  make([]context.Context, size+1),
		w:    0,
		r:    0,
		size: size,
		len:  size + 1,
		wait: make(map[chan bool]interface{}),
	}
	return q
}

func (q *Queue) Push(ctx context.Context) bool {
	defer q.mtx.Unlock()
	q.mtx.Lock()
	if ctx == nil {
		return false
	}
	w := q.w
	if q.len > (w + 1) {
		w++
	} else {
		w = 0
	}
	if q.r == w {
		return false
	}
	q.buf[q.w] = ctx
	q.w = w

	q._wakeup(nil)
	return true
}

func (q *Queue) Pop() context.Context {
	defer q.mtx.Unlock()
	q.mtx.Lock()

	if q.w == q.r {
		return nil
	}
	ctx := q.buf[q.r].(context.Context)
	q.buf[q.r] = nil
	if q.len > (q.r + 1) {
		q.r++
	} else {
		q.r = 0
	}
	return ctx
}

func (q *Queue) _wait() chan bool {
	defer q.mtxWait.Unlock()
	q.mtxWait.Lock()
	ch := make(chan bool)
	q.wait[ch] = true
	return ch
}
func (q *Queue) _wakeup(ch chan bool) {
	defer q.mtxWait.Unlock()
	q.mtxWait.Lock()
	if ch == nil {
		for k := range q.wait {
			ch = k
			break
		}
	}
	if ch != nil {
		close(ch)
		delete(q.wait, ch)
	}
}

func (q *Queue) Wait() <-chan bool {
	defer q.mtx.RUnlock()
	q.mtx.RLock()
	ch := q._wait()
	if q.w != q.r {
		q._wakeup(ch)
	}
	return ch
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
