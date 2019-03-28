package chain

import (
	"log"
	"sync"
	"time"
)

type txExecutionEntry struct {
	count     int
	execution time.Duration
	finalize  time.Duration
}

func (e *txExecutionEntry) Sub(e2 *txExecutionEntry) {
	e.count -= e2.count
	e.execution -= e2.execution
	e.finalize -= e2.finalize
}

func (e *txExecutionEntry) Add(e2 *txExecutionEntry) {
	e.count += e2.count
	e.execution += e2.execution
	e.finalize += e2.finalize
}

type regulator struct {
	lock sync.Mutex

	commitTimeout time.Duration

	history      [30]txExecutionEntry
	sum          txExecutionEntry
	currentIndex int

	currentTxCount int
}

func (r *regulator) SetCommitTimeout(d time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.commitTimeout == d {
		return
	}
	log.Printf("Regulator.SetCommitTimeout(%s)", d)

	txCount := int(d * time.Duration(r.currentTxCount) / r.commitTimeout)
	r.commitTimeout = d
	r.currentTxCount = txCount
}

func (r *regulator) CommitTimeout() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.commitTimeout
}

func (r *regulator) MaxTxCount() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.currentTxCount
}

func (r *regulator) OnTxExecution(count int, ed time.Duration, fd time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if count == 0 {
		return
	}

	e := txExecutionEntry{count, ed, fd}
	item := &r.history[r.currentIndex]
	r.sum.Sub(item)
	*item = e
	r.sum.Add(&e)
	r.currentIndex = (r.currentIndex + 1) % (len(r.history))

	// For target duration
	r.currentTxCount =
		int(time.Duration(r.sum.count) * r.commitTimeout / (r.sum.execution + (r.sum.finalize * 2)))
	log.Printf("OnTxExecution: TxCount=%d Execution=%s Finalize=%s -> MaxTxCount=%d",
		count, ed, fd, r.currentTxCount)
}

func NewRegulator(duration time.Duration, count int) *regulator {
	return &regulator{
		commitTimeout:  duration,
		currentTxCount: count,
	}
}
