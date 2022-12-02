package network

import (
	"context"
	"sync"
)

type Queue interface {
	Push(ctx context.Context) bool
	Pop() context.Context
	Wait() <-chan bool
	Available() int
	Close()
}

type ChannelQueue struct {
	buffer chan context.Context
}

func (q *ChannelQueue) Push(c context.Context) bool {
	select {
	case q.buffer <- c:
		return true
	default:
		return false
	}
}

func (q *ChannelQueue) Wait() <-chan context.Context {
	return q.buffer
}

func (q *ChannelQueue) Pop() context.Context {
	select {
	case c := <-q.buffer:
		return c
	default:
		return nil
	}
}

func newChannelQueue(size int) *ChannelQueue {
	return &ChannelQueue{
		buffer: make(chan context.Context, size),
	}
}

type sliceQueue struct {
	buffer      []context.Context
	read, write int
	size, len   int
}

func (q *sliceQueue) init(size int) {
	q.size = size
	// q.buffer = make([]context.Context, size)
}

func (q *sliceQueue) push(c context.Context) bool {
	if q.len == q.size {
		return false
	}
	if q.buffer == nil {
		q.buffer = make([]context.Context, q.size)
	}
	q.buffer[q.write] = c
	q.len += 1
	q.write = (q.write + 1) % q.size
	return true
}

func (q *sliceQueue) pop() (context.Context, bool) {
	if q.len < 1 {
		return nil, false
	}
	v := q.buffer[q.read]
	q.buffer[q.read] = nil
	q.len -= 1
	q.read = (q.read + 1) % q.size
	return v, true
}

func (q *sliceQueue) available() int {
	return q.size - q.len
}

type singleQueue struct {
	sliceQueue

	lock sync.Mutex
	out  chan bool
}

func (q *singleQueue) notify() {
	select {
	case q.out <- true:
	default:
	}
}

func (q *singleQueue) Push(c context.Context) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	r := q.push(c)
	if r && q.len == 1 {
		q.notify()
	}
	return r
}

func (q *singleQueue) Pop() context.Context {
	q.lock.Lock()
	defer q.lock.Unlock()

	ctx, ok := q.pop()
	if ok && q.len > 0 {
		q.notify()
	}
	return ctx
}

func (q *singleQueue) Available() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.size - q.len
}

func (q *singleQueue) Wait() <-chan bool {
	return q.out
}

func (q *singleQueue) Close() {
	close(q.out)
}

func NewQueue(size int) Queue {
	q := new(singleQueue)
	q.init(size)
	q.out = make(chan bool, 1)
	return q
}

type multiQueue struct {
	queues []sliceQueue
	len    int

	lock      sync.Mutex
	out       chan bool
	fetchFunc func() (context.Context, bool)
}

func (q *multiQueue) init(size int, cnt int) {
	queues := make([]sliceQueue, cnt)
	for i := 0; i < cnt; i++ {
		queues[i].init(size)
	}
	q.queues = queues
	q.out = make(chan bool, 1)
}

func (q *multiQueue) Push(c context.Context, idx int) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	if idx < 0 || idx >= len(q.queues) {
		return false
	}
	if ok := q.queues[idx].push(c); !ok {
		return false
	}
	q.len += 1
	q.notify()
	return true
}

func (q *multiQueue) notify() {
	select {
	case q.out <- true:
	default:
	}
}

func (q *multiQueue) term() {
	close(q.out)
}

func (q *multiQueue) Pop() context.Context {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.len < 1 {
		return nil
	}

	ctx, ok := q.fetchFunc()
	if ok {
		q.len -= 1
		if q.len > 0 {
			q.notify()
		}
	}
	return ctx
}

func (q *multiQueue) Wait() <-chan bool {
	return q.out
}

func (q *multiQueue) Available(idx int) int {
	q.lock.Lock()
	defer q.lock.Unlock()
	if idx < 0 || idx >= len(q.queues) {
		return 0
	}
	return q.queues[idx].available()
}

type PriorityQueue struct {
	multiQueue
}

func (q *PriorityQueue) fetch() (context.Context, bool) {
	for i := 0; i < len(q.queues); i++ {
		if ctx, ok := q.queues[i].pop(); ok {
			return ctx, true
		}
	}
	return nil, false
}

func (q *PriorityQueue) Close() {
	q.term()
}

func NewPriorityQueue(size int, maxPriority int) *PriorityQueue {
	q := &PriorityQueue{}
	q.init(size, maxPriority+1)
	q.fetchFunc = q.fetch
	return q
}

type WeightQueue struct {
	multiQueue
	weights []int
	current []int
	idx     int
}

func (q *WeightQueue) fetch() (context.Context, bool) {
	s := len(q.queues)
	for i := 0; i < s; i++ {
		idx := (q.idx + i) % s
		if ctx, ok := q.queues[idx].pop(); ok {
			q.current[idx] += 1
			if q.current[idx] >= q.weights[idx] {
				q.current[idx] = 0
				idx = (idx + 1) % s
			}
			q.idx = idx
			return ctx, true
		} else {
			q.current[idx] = 0
		}
	}
	return nil, false
}

func (q *WeightQueue) SetWeight(idx int, weight int) error {
	if idx < 0 || idx >= len(q.weights) || weight < 1 {
		return ErrIllegalArgument
	}
	q.weights[idx] = weight
	return nil
}

func (q *WeightQueue) Close() {
	q.term()
}

func NewWeightQueue(size int, nq int) *WeightQueue {
	q := new(WeightQueue)
	q.init(size, nq)
	q.fetchFunc = q.fetch
	q.weights = make([]int, nq)
	for i := 0; i < nq; i++ {
		q.weights[i] = 1
	}
	q.current = make([]int, nq)
	return q
}
