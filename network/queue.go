package network

import (
	"context"
	"sync"

	"github.com/go-errors/errors"
)

type Queue interface {
	Push(ctx context.Context) bool
	Pop() context.Context
	Wait() <-chan bool
	Clear()
	Available() int
	IsEmpty() bool
	Size() int
}

type queue struct {
	buf     []context.Context
	w       int
	r       int
	len     int
	size    int
	mtx     sync.RWMutex
	wait    map[chan bool]interface{}
	mtxWait sync.Mutex
}

func NewQueue(size int) Queue {
	if size < 1 {
		panic("queue size must be greater than zero")
	}
	q := &queue{
		buf:  make([]context.Context, size+1),
		w:    0,
		r:    0,
		size: size,
		len:  size + 1,
		wait: make(map[chan bool]interface{}),
	}
	return q
}

func (q *queue) Push(ctx context.Context) bool {
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

func (q *queue) Pop() context.Context {
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

func (q *queue) _wait() chan bool {
	defer q.mtxWait.Unlock()
	q.mtxWait.Lock()
	ch := make(chan bool)
	q.wait[ch] = true
	return ch
}
func (q *queue) _wakeup(ch chan bool) bool {
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
		return true
	}
	return false
}

func (q *queue) Wait() <-chan bool {
	defer q.mtx.RUnlock()
	q.mtx.RLock()
	ch := q._wait()
	if q.w != q.r {
		q._wakeup(ch)
	}
	return ch
}

func (q *queue) Clear() {
	defer q.mtx.Unlock()
	q.mtx.Lock()
	for q._wakeup(nil) {
	}
	q.r = 0
	q.w = 0
	q.buf = make([]context.Context, q.size+1)
}

func (q *queue) Available() int {
	defer q.mtx.RUnlock()
	q.mtx.RLock()
	if q.w < q.r {
		return q.size - q.r + q.w
	}
	return q.w - q.r
}
func (q *queue) IsEmpty() bool {
	defer q.mtx.RUnlock()
	q.mtx.RLock()
	return q.w == q.r
}

func (q *queue) Size() int {
	return q.size
}

var (
	ErrInvalidArgument = errors.New("invalid argument")
)

type MultiQueue struct {
	s       []Queue
	size    int
	mtx     sync.RWMutex
	wait    map[chan bool]interface{}
	mtxWait sync.Mutex
}

func NewMultiQueue(size int, numberOfQueue int) *MultiQueue {
	if size < 1 {
		panic("queue size must be greater than zero")
	}
	if numberOfQueue < 1 {
		panic("number Of queue must be greater than zero")
	}

	mq := &MultiQueue{
		s:    make([]Queue, numberOfQueue),
		size: numberOfQueue,
		wait: make(map[chan bool]interface{}),
	}
	for i := 0; i < numberOfQueue; i++ {
		mq.s[i] = NewQueue(size)
	}
	return mq
}

func (mq *MultiQueue) Push(ctx context.Context, queueIndex int) bool {
	defer mq.mtx.Unlock()
	mq.mtx.Lock()

	if queueIndex >= mq.size || queueIndex < 0 {
		return false
	}

	q := mq.s[queueIndex]
	if !q.Push(ctx) {
		return false
	}

	mq._wakeup(nil)
	return true
}

func (mq *MultiQueue) Pop(queueIndex int) context.Context {
	defer mq.mtx.Unlock()
	mq.mtx.Lock()

	if queueIndex >= mq.size || queueIndex < 0 {
		return nil
	}

	q := mq.s[queueIndex]
	return q.Pop()
}

func (mq *MultiQueue) _wait() chan bool {
	defer mq.mtxWait.Unlock()
	mq.mtxWait.Lock()

	ch := make(chan bool)
	mq.wait[ch] = true
	return ch
}
func (mq *MultiQueue) _wakeup(ch chan bool) bool {
	defer mq.mtxWait.Unlock()
	mq.mtxWait.Lock()
	if ch == nil {
		for k := range mq.wait {
			ch = k
			break
		}
	}
	if ch != nil {
		close(ch)
		delete(mq.wait, ch)
		return true
	}
	return false
}

func (mq *MultiQueue) _empty() bool {
	for _, q := range mq.s {
		if !q.IsEmpty() {
			return false
		}
	}
	return true
}

func (mq *MultiQueue) Wait() <-chan bool {
	defer mq.mtx.RUnlock()
	mq.mtx.RLock()
	ch := mq._wait()

	if !mq._empty() {
		mq._wakeup(ch)
	}
	return ch
}

func (mq *MultiQueue) _clear() {
	for mq._wakeup(nil) {
	}
	for _, q := range mq.s {
		q.Clear()
	}
}

func (mq *MultiQueue) Clear() {
	defer mq.mtx.Unlock()
	mq.mtx.Lock()
	mq._clear()
}

func (mq *MultiQueue) Available(queueIndex int) int {
	defer mq.mtx.RUnlock()
	mq.mtx.RLock()

	if queueIndex >= mq.size || queueIndex < 0 {
		return -1
	}

	q := mq.s[queueIndex]
	return q.Available()
}

func (mq *MultiQueue) IsEmpty() bool {
	defer mq.mtx.RUnlock()
	mq.mtx.RLock()
	return mq._empty()
}
func (mq *MultiQueue) Len() int {
	return mq.size
}


type WeightQueue struct {
	*MultiQueue
	w           []int
	t           []int
}

func NewWeightQueue(size int, numberOfQueue int) *WeightQueue {
	wq := &WeightQueue{
		MultiQueue: NewMultiQueue(size, numberOfQueue),
		w:           make([]int, numberOfQueue),
		t:           make([]int, numberOfQueue),
	}
	for i := 0; i < numberOfQueue; i++ {
		wq.w[i] = 1
		wq.t[i] = 1
	}
	return wq
}

func (wq *WeightQueue) SetWeight(queueIndex int, weight int) error {
	defer wq.mtx.Unlock()
	wq.mtx.Lock()

	if queueIndex >= wq.size || queueIndex < 0 || weight < 1 {
		return ErrInvalidArgument
	}

	wq.w[queueIndex] = weight
	wq.t[queueIndex] = weight
	return nil
}

func (wq *WeightQueue) _pop() context.Context {
	for i, q := range wq.s {
		if t := wq.t[i]; t > 0 {
			if ctx := q.Pop(); ctx != nil {
				wq.t[i]--
				return ctx
			}
		}
	}
	return nil
}

func (wq *WeightQueue) Pop() context.Context {
	defer wq.mtx.Unlock()
	wq.mtx.Lock()

	if ctx := wq._pop(); ctx != nil {
		return ctx
	}
	for i, w := range wq.w {
		wq.t[i] = w
	}
	return wq._pop()
}

func (wq *WeightQueue) Clear() {
	defer wq.mtx.Unlock()
	wq.mtx.Lock()

	wq.MultiQueue._clear()
	for i, w := range wq.w {
		wq.t[i] = w
	}
}

type PriorityQueue struct {
	*MultiQueue
	maxPriority uint8
}

func NewPriorityQueue(size int, maxPriority uint8) *PriorityQueue {
	if maxPriority < 0 {
		panic("max priority must be positive number")
	}
	l := int(maxPriority) + 1
	pq := &PriorityQueue{
		MultiQueue: NewMultiQueue(size, l),
		maxPriority: maxPriority,
	}
	return pq
}

func (pq *PriorityQueue) Push(ctx context.Context, priority uint8) bool {
	return pq.MultiQueue.Push(ctx, int(priority))
}

func (pq *PriorityQueue) Pop() context.Context {
	defer pq.mtx.Unlock()
	pq.mtx.Lock()

	for _, q := range pq.s {
		if ctx := q.Pop(); ctx != nil {
			return ctx
		}
	}
	return nil
}

