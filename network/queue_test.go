package network

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_queue(t *testing.T) {
	q := NewQueue(2)
	for i := 0; i < q.Size(); i++ {
		ctx := context.WithValue(context.Background(), "i", i)
		assert.True(t, q.Push(ctx), "true")
	}
	assert.Equal(t, q.Size(), q.Available(), "size")

	ctx := context.WithValue(context.Background(), "i", q.Size())
	assert.False(t, q.Push(ctx), "false")

	select {
	case <-q.Wait():
		for i := 0; i < q.Size(); i++ {
			ctx := q.Pop()
			if ctx == nil {
				assert.FailNow(t, "queue pop fail")
			}
			ri := ctx.Value("i").(int)
			assert.Equal(t, i, ri, "sequence fail")
		}
		assert.Nil(t, q.Pop(), "nil")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "queue wait fail", "when has elements")
	}

	var wg sync.WaitGroup
	wg.Add(2)
	st := make(chan bool)

	l := q.Size() * 2
	var push int
	go func() {
		<-st
		for ; push < l; push++ {
			ctx := context.WithValue(context.Background(), "i", push)
			if !q.Push(ctx) {
				time.Sleep(time.Millisecond)
				assert.True(t, q.Push(ctx), "true")
			}
		}
		wg.Done()
	}()

	var pop int
	go func() {
		<-st
	Loop:
		for ; pop < l; {
			select {
			case <-q.Wait():
			LoopWait:
				for {
					ctx := q.Pop()
					if ctx == nil {
						break LoopWait
					}
					ri := ctx.Value("i").(int)
					assert.Equal(t, pop, ri, "sequence fail")
					pop++
				}
			case <-time.After(time.Second):
				t.Log("Timeout")
				break Loop
			}
		}
		wg.Done()
	}()

	close(st)
	wg.Wait()
	assert.Equal(t, l, pop, "pop")
}

func Test_queue_WeightQueue(t *testing.T) {
	q := NewWeightQueue(10, 2)
	assert.NoError(t, q.SetWeight(1, 2), "NoError q.SetWeight")
	for qi := 0; qi < q.NumberOfQueue(); qi++ {
		w := q.Weight(qi)
		for i := 0; i < q.Size(); i++ {
			ctx := context.WithValue(context.Background(), "qi", qi)
			ctx = context.WithValue(ctx, "w", w)
			ctx = context.WithValue(ctx, "i", i)
			assert.True(t, q.Push(ctx, qi), "true")
		}
		assert.Equal(t, q.Size(), q.Available(qi), "size")
		ctx := context.WithValue(context.Background(), "qi", qi)
		ctx = context.WithValue(ctx, "w", w)
		ctx = context.WithValue(ctx, "i", q.Size())
		assert.False(t, q.Push(ctx, qi), "false")
	}

	select {
	case <-q.Wait():
		li := make([]int, q.NumberOfQueue())
		tn := q.Size() * q.NumberOfQueue()
		qi := 0
		w := q.Weight(0)
		wi := 0
		for i := 0; i < tn; i++ {
			ctx := q.Pop()
			if ctx == nil {
				assert.FailNow(t, "queue pop fail")
			}
			rqi := ctx.Value("qi").(int)
			rw := ctx.Value("w").(int)
			assert.Equal(t, q.Weight(rqi), rw, "weight")
			ri := ctx.Value("i").(int)
			assert.Equal(t, li[rqi], ri, "sequence fail")
			li[rqi] = ri + 1
			if qi != rqi {
				assert.Equal(t, w, wi, "weight fail")
				qi = rqi
				w = rw
				wi = 0
			}
			w = li[rqi] % w
			if w == 0 {
				w = rw
			}
			wi++
		}
		assert.Nil(t, q.Pop(), "nil")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "queue wait fail", "when has elements")
	}
}

func Test_queue_PriorityQueue(t *testing.T) {
	q := NewPriorityQueue(10, 1)
	for p := q.MaxPriority(); p >= 0; p-- {
		for i := 0; i < q.Size(); i++ {
			ctx := context.WithValue(context.Background(), "p", p)
			ctx = context.WithValue(ctx, "i", i)
			assert.True(t, q.Push(ctx, uint8(p)), "true")
		}
		assert.Equal(t, q.Size(), q.Available(p), "size")
		ctx := context.WithValue(context.Background(), "p", p)
		ctx = context.WithValue(ctx, "i", q.Size())
		assert.False(t, q.Push(ctx, uint8(p)), "false")
	}

	select {
	case <-q.Wait():
		for p := 0; p <= q.MaxPriority(); p++ {
			for i := 0; i < q.Size(); i++ {
				ctx := q.Pop()
				if ctx == nil {
					assert.FailNow(t, "queue pop fail")
				}
				rp := ctx.Value("p").(int)
				ri := ctx.Value("i").(int)
				assert.Equal(t, p, rp, "priority fail")
				assert.Equal(t, i, ri, "priority fail")
			}
		}
		assert.Nil(t, q.Pop(), "nil")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "queue wait fail", "when has elements")
	}
}
