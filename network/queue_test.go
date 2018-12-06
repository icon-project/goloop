package network

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func generateOnReceiveContext(s string, i int) context.Context {
	pkt := generateDummyPacket(s, i)
	p := generateDummyPeer(s)
	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
	return ctx
}

func Test_queue_OnReceiveQueue(t *testing.T) {
	q := NewQueue(10)
	assert.Equal(t, 0, q.Available(), "0")
	assert.Equal(t, false, q.Push(nil), "false")

	var ctxSize = 4
	var workerSize = 2

	arr := make([]context.Context, ctxSize)
	for i := range arr {
		s := fmt.Sprintf("%d", i)
		arr[i] = generateOnReceiveContext(s, i)
		assert.Equal(t, true, q.Push(arr[i]), "true")
		assert.Equal(t, i+1, q.Available(), "Available")
	}

	for _, v := range arr {
		<-q.Wait()
		ctx := q.Pop()
		assert.Equal(t, v, ctx, "Queue.Pop")
	}

	ch := make(chan context.Context, ctxSize)
	var wg sync.WaitGroup
	for i := 0; i < workerSize; i++ {
		wg.Add(1)
		go func(i int, q *Queue, ch chan<- context.Context) {
			log.Println(i, "Queue.Wait")
			for {
				select {
				case <-q.Wait():
					for {
						ctx := q.Pop()
						if ctx == nil {
							log.Println(i, "Queue.Empty")
							break
						}
						ch <- ctx
						pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
						log.Println(i, "Queue.Pop", string(pkt.payload))
						time.Sleep(1 * time.Millisecond)
					}
				case <-time.After(3 * time.Second):
					wg.Done()
				}
			}
		}(i, q, ch)
	}

	go func() {
		for i := range arr {
			assert.Equal(t, true, q.Push(arr[i]), "true")
			assert.Equal(t, i+1, q.Available(), fmt.Sprint(i+1))
		}
	}()

	for _, v := range arr {
		ctx := <-ch
		assert.Equal(t, v, ctx, "Queue.Pop")
	}

	log.Println("WaitGroup.Wait")
	wg.Wait()
	log.Println("finish")
}

func Benchmark_queue_OnReceiveQueue(b *testing.B) {
	b.StopTimer()
	q := NewQueue(10000)
	arr := make([]context.Context, b.N)
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%d", i)
		arr[i] = generateOnReceiveContext(s, i)
	}

	pushAndPop := true
	if !pushAndPop {
		go func(q *Queue, n int) {
			var t = 0
			for {
				<-q.Wait()
				for {
					if ctx := q.Pop(); ctx == nil {
						break
					}
					t++
				}
				if n == t {
					break
				}
			}
		}(q, b.N)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx := arr[i]
		if !q.Push(ctx) {
			b.Fail()
		}
		if pushAndPop {
			if q.Pop() == nil {
				b.Fail()
			}
		}
	}
}

func Benchmark_dummy_OnReceiveContext(b *testing.B) {
	arr := make([]context.Context, b.N)
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%d", i)
		arr[i] = generateOnReceiveContext(s, i)
	}
	//Benchmark_dummy_OnReceiveContext-8   	 1000000	      1003 ns/op	     592 B/op	      11 allocs/op
}
