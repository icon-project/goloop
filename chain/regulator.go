package chain

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common/log"
)

const (
	configMinimumTransactions     = 10
	ConfigDefaultTransactions     = 1000
	ConfigDefaultCommitTimeout    = time.Second
	ConfigDefaultBlockInterval    = time.Second
	ConfigDefaultMinCommitTimeout = 200 * time.Millisecond
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

	proposeTime      time.Time
	blockInterval    time.Duration
	minCommitTimeout time.Duration

	history      [30]txExecutionEntry
	sum          txExecutionEntry
	currentIndex int

	currentTxCount int

	log log.Logger
}

func (r *regulator) SetBlockInterval(blockInterval time.Duration, commitTimeout time.Duration) {
	if blockInterval == 0 {
		blockInterval = commitTimeout
	} else if commitTimeout == 0 {
		commitTimeout = ConfigDefaultMinCommitTimeout
	}
	if commitTimeout > blockInterval {
		commitTimeout = blockInterval
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if r.minCommitTimeout == commitTimeout && r.blockInterval == blockInterval {
		return
	}

	r.log.Printf("Regulator.SetCommitTimeout(interval=%s,timeout=%s)", blockInterval, commitTimeout)

	txCount := int(blockInterval * time.Duration(r.currentTxCount) / r.blockInterval)
	r.blockInterval = blockInterval
	r.minCommitTimeout = commitTimeout
	r.currentTxCount = txCount
}

func (r *regulator) OnPropose(now time.Time) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.proposeTime = now
}

func (r *regulator) CommitTimeout() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	timeout := r.blockInterval - time.Now().Sub(r.proposeTime)
	if timeout < r.minCommitTimeout {
		timeout = r.minCommitTimeout
	}

	return timeout
}

func (r *regulator) MinCommitTimeout() time.Duration {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.minCommitTimeout
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
		int(time.Duration(r.sum.count) * r.blockInterval / (r.sum.execution + (r.sum.finalize * 2)))
	if r.currentTxCount < configMinimumTransactions {
		r.currentTxCount = configMinimumTransactions
	}
	r.log.Printf("OnTxExecution: TxCount=%d Execution=%s Finalize=%s -> MaxTxCount=%d",
		count, ed, fd, r.currentTxCount)
}

func NewRegulator(logger log.Logger) *regulator {
	return &regulator{
		blockInterval:    ConfigDefaultBlockInterval,
		minCommitTimeout: ConfigDefaultMinCommitTimeout,
		currentTxCount:   ConfigDefaultTransactions,
		log:              logger,
	}
}
