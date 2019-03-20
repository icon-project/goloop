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

func Test_queue(t *testing.T) {
	q := NewQueue(10)
	ticker := time.NewTicker(100 * time.Millisecond)
	go func(q Queue) {
		for i := 0; i < 11; i++ {
			ctx := context.WithValue(context.Background(), "i", i)
			r := q.Push(ctx)
			log.Println("push", i, r)
		}
	}(q)
	n := 0
Loop:
	for {
		select {
		case <-q.Wait():
			log.Println("wakeup")
			for {
				ctx := q.Pop()
				if ctx == nil {
					break
				}
				i := ctx.Value("i").(int)
				log.Println("preSleep", i)
				time.Sleep(500 * time.Millisecond)
				log.Println("afterSleep", i)
			}
			log.Println("wakeup done")
		case <-ticker.C:
			n++
			log.Println("ticker", n)
			if n%10 == 0 {
				ctx := context.WithValue(context.Background(), "i", n)
				q.Push(ctx)
			}
			if n > 20 {
				break Loop
			}
		}
	}
}

func Test_queue_WeightQueue(t *testing.T) {
	var wg sync.WaitGroup
	q := NewWeightQueue(10, 2)
	assert.NoError(t,q.SetWeight(1, 2),"NoError q.SetWeight")
	ticker := time.NewTicker(300 * time.Millisecond)
	wg.Add(11)
	go func(q *WeightQueue, n int, p int) {
		for i := 0; i < n; i++ {
			ctx := context.WithValue(context.Background(), "i", i)
			ctx = context.WithValue(ctx, "p", p)
			r := q.Push(ctx, p)
			if !r {
				wg.Add(-1)
			}
			log.Println("push p:", p, "i:", i, "r:", r)
		}
	}(q, 11, 1)


	ch := make(chan bool)
	go func(q *WeightQueue) {
	Loop:
		for {
			select {
			case <-q.Wait():
				log.Println("wakeup")
				for {
					ctx := q.Pop()
					if ctx == nil {
						log.Println("ctx nil")
						break
					}
					i := ctx.Value("i").(int)
					p := ctx.Value("p").(int)
					log.Println("pop p:", p, "i:", i)
					switch p {
					case 0:
						time.Sleep(100 * time.Millisecond)
					case 1:
						time.Sleep(200 * time.Millisecond)
					}
					wg.Done()
				}
				log.Println("wakeup done")
			case <-ch:
				break Loop
			}
		}
	}(q)

	go func(ch chan bool) {
		wg.Wait()
		close(ch)
	}(ch)

	n := 0
Loop:
	for {
		select {
		case <-ticker.C:
			ctx := context.WithValue(context.Background(), "i", n)
			ctx = context.WithValue(ctx, "p", 0)
			wg.Add(1)
			r := q.Push(ctx, 0)
			if !r {
				wg.Add(-1)
			}
			log.Println("push p:", 0, "i:", n, "r:", r)
			n++
		case <-ch:
			break Loop
		}
	}
}

func Test_queue_PriorityQueue(t *testing.T) {
	var wg sync.WaitGroup
	q := NewPriorityQueue(10, 1)
	ticker := time.NewTicker(300 * time.Millisecond)
	wg.Add(11)
	go func(q *PriorityQueue, n int, p uint8) {
		for i := 0; i < n; i++ {
			ctx := context.WithValue(context.Background(), "i", i)
			ctx = context.WithValue(ctx, "p", int(p))
			r := q.Push(ctx, p)
			if !r {
				wg.Add(-1)
			}
			log.Println("push p:", p, "i:", i, "r:", r)
		}
	}(q, 11, 1)


	ch := make(chan bool)
	go func(q *PriorityQueue) {
	Loop:
		for {
			select {
			case <-q.Wait():
				log.Println("wakeup")
				for {
					ctx := q.Pop()
					if ctx == nil {
						log.Println("ctx nil")
						break
					}
					i := ctx.Value("i").(int)
					p := ctx.Value("p").(int)
					log.Println("pop p:", p, "i:", i)
					switch p {
					case 0:
						time.Sleep(100 * time.Millisecond)
					case 1:
						time.Sleep(200 * time.Millisecond)
					}
					wg.Done()
				}
				log.Println("wakeup done")
			case <-ch:
				break Loop
			}
		}
	}(q)

	go func(ch chan bool) {
		wg.Wait()
		close(ch)
	}(ch)

	n := 0
Loop:
	for {
		select {
		case <-ticker.C:
			ctx := context.WithValue(context.Background(), "i", n)
			ctx = context.WithValue(ctx, "p", 0)
			wg.Add(1)
			r := q.Push(ctx, 0)
			if !r {
				wg.Add(-1)
			}
			log.Println("push p:", 0, "i:", n, "r:", r)
			n++
		case <-ch:
			break Loop
		}
	}
}

func Test_queue_OnReceiveQueue(t *testing.T) {
	q := NewQueue(10)
	assert.Equal(t, 0, q.Available(), "0")
	assert.False(t, q.Push(nil), "false")

	var ctxSize = 4
	var workerSize = 2

	arr := make([]context.Context, ctxSize)
	for i := range arr {
		s := fmt.Sprintf("%d", i)
		arr[i] = generateOnReceiveContext(s, i)
		assert.True(t, q.Push(arr[i]), "true")
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
		go func(i int, q Queue, ch chan<- context.Context) {
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
			assert.True(t, q.Push(arr[i]), "true")
			log.Println(i, "Queue.Push")
			//assert.Equal(t, i+1, q.Available(), fmt.Sprint(i+1))
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
		go func(q Queue, n int) {
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
