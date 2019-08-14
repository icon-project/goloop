package network

import (
	"context"
	"log"
	"sync"
	"testing"
)

func TestPriorityQueue_Pop(t *testing.T) {
	priorities := 4
	rounds := 20
	q := NewPriorityQueue(rounds, priorities-1)

	var exit sync.WaitGroup
	exit.Add(1)
	var event sync.WaitGroup
	go func() {
		closed := false
		log.Println("WAIT for items")
		for !closed {
			select {
			case _, ok := <-q.Wait():
				if !ok {
					closed = true
					log.Println("Queue CLOSED")
					break
				}
				ctx := q.Pop()
				if ctx != nil {
					log.Printf("ITEM(priority=%d,round=%d)",
						ctx.Value("priority"),
						ctx.Value("round"))
					event.Done()
				} else {
					t.FailNow()
					return
				}
			}
		}
		exit.Done()
	}()

	log.Println("SEND items to the queue")
	for i := 0; i < rounds; i++ {
		for p := 0; p < priorities; p++ {
			event.Add(1)
			ctx := context.WithValue(context.Background(), "priority", int(p))
			ctx = context.WithValue(ctx, "round", int(i))
			q.Push(ctx, p)
		}
	}
	log.Println("WAIT util all items are received")
	event.Wait()
	log.Println("CLOSE Queue")
	q.Close()
	exit.Wait()
}

func TestWeightQueue_Pop(t *testing.T) {
	queueCount := 4
	rounds := 20
	q := NewWeightQueue(rounds, queueCount)
	q.SetWeight(0, 2)

	var exit sync.WaitGroup
	exit.Add(1)
	var event sync.WaitGroup
	go func() {
		closed := false
		log.Println("WAIT for items")
		for !closed {
			select {
			case _, ok := <-q.Wait():
				if !ok {
					closed = true
					log.Println("Queue CLOSED")
					break
				}
				ctx := q.Pop()
				if ctx != nil {
					log.Printf("ITEM(index=%d,round=%d)",
						ctx.Value("index"),
						ctx.Value("round"))
					event.Done()
				} else {
					t.FailNow()
					return
				}
			}
		}
		exit.Done()
	}()

	log.Println("SEND items to the queue")
	for i := 0; i < rounds; i++ {
		for idx := 0; idx < queueCount; idx++ {
			event.Add(1)
			ctx := context.WithValue(context.Background(), "index", int(idx))
			ctx = context.WithValue(ctx, "round", int(i))
			q.Push(ctx, idx)
		}
	}
	log.Println("WAIT util all items are received")
	event.Wait()
	log.Println("CLOSE Queue")
	q.Close()
	exit.Wait()
}
