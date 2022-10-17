package sync2

import (
	"fmt"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
)

const (
	configPackSize   int = 50
	configRoundLimit int = 500
)

type SyncProcessor interface {
	Start(cb func(err error))
	Stop()
	AddRequest(id db.BucketID, key []byte) error
	UnresolvedCount() int
}

type syncProcessor struct {
	mutex  sync.Mutex
	waiter *sync.Cond
	logger log.Logger

	builder  merkle.Builder
	reactors []SyncReactor

	// reactor ready+sent+checked
	// ready --> sent -(no valid)-> checked -(no ready/sent) -> ready
	//                -(valid)----> ready
	readyPool   *peerPool
	sentPool    *peerPool
	checkedPool *peerPool

	timer      *time.Timer
	datasyncer bool
	migrateDur time.Duration

	reqIter  merkle.RequestIterator
	reqCount int
}

func (s *syncProcessor) onTermInLock() {
	s.logger.Infoln("onTermInLock()")

	s.stopTimerInLock()
	s.readyPool = nil
	s.sentPool = nil
	s.checkedPool = nil
}

func (s *syncProcessor) OnPeerJoin(p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.readyPool == nil {
		return
	}

	s.readyPool.push(p)
}

func (s *syncProcessor) OnPeerLeave(p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.readyPool == nil || s.sentPool == nil || s.checkedPool == nil {
		return
	}

	if p2 := s.readyPool.remove(p.id); p2 != nil {
		s.onPoolChangeInLock()
		return
	}
	if p2 := s.sentPool.remove(p.id); p2 != nil {
		s.onPoolChangeInLock()
		return
	}
	if p2 := s.checkedPool.remove(p.id); p2 != nil {
		return
	}
}

func (s *syncProcessor) onInitInLock() {
	// init pool migrate duration
	if s.datasyncer {
		s.migrateDur = configDataSyncMigrationInterval * time.Second
	} else {
		s.migrateDur = configMigrationInterval * time.Second
	}

	// init readyPool
	for _, reactor := range s.reactors {
		pList := reactor.WatchPeers(s)
		for _, p := range pList {
			s.readyPool.push(p)
		}
	}
}

func (s *syncProcessor) Start(cb func(err error)) {
	go s.run(cb)
}

func (s *syncProcessor) run(cb func(err error)) {
	var err error

	defer func() {
		cb(err)
	}()

	err = s.DoSync()
}

func (s *syncProcessor) stopTimerInLock() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *syncProcessor) DoSync() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.onInitInLock()

	var err error
	for {
		if s.builder == nil {
			err = errors.ErrInterrupted
			s.logger.Infof("DoSync() stop syncProcessor by %v", err)
			break
		}

		count := s.builder.UnresolvedCount()
		s.logger.Tracef("DoSync() unresolvedCount=%d", count)

		if count == 0 && !s.datasyncer {
			s.logger.Infof("DoSync() done syncProcessor")
			break
		}

		s.logger.Tracef("DoSync() readyPool=%d, sentPool=%d", s.readyPool.size(), s.sentPool.size())
		for count > 0 && s.readyPool.size() > 0 {
			if s.sendRequestsInLock() {
				break
			}
			s.logger.Tracef("DoSync() retry sendRequest. readyPool=%d", s.readyPool.size())
		}

		s.logger.Tracef("DoSync() waiting signal. unresolvedCount=%d, readyPool=%d, sentPool=%d",
			count, s.readyPool.size(), s.sentPool.size())
		s.waiter.Wait()
	}

	s.onTermInLock()
	return err
}

// Stop sync processor
func (s *syncProcessor) Stop() {
	s.logger.Infoln("Stop() sync processor")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.builder = nil
	s.wakeupInLock()
}

func (s *syncProcessor) AddRequest(id db.BucketID, key []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.builder != nil {
		if id.Hasher() == nil {
			return errors.IllegalArgumentError.Errorf("InvalidBucket(id=%q)", id)
		}
		bk, err := s.builder.Database().GetBucket(id)
		if err != nil {
			return err
		}
		if value, err := bk.Get(key); err != nil {
			return err
		} else if value != nil {
			return nil
		}
		s.logger.Debugf("AddRequest() REQUEST id=%s key=%#x", id, key)
		s.builder.RequestData(id, key, onDataHandler(func() {
			s.logger.Debugf("AddRequest() ADD id=%s key=%#x", id, key)
		}))

		s.wakeupInLock()
		return err
	} else {
		return errors.InvalidStateError.Errorf("Terminated")
	}
}

func (s *syncProcessor) UnresolvedCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.builder.UnresolvedCount()
}

// sendRequestsInLock
//
// returns true if succeed sendRequest, false if no send.
// syncProcessor --> peer --> PeerHandler(Reactor) --> module.ProtocolHandler
func (s *syncProcessor) sendRequestsInLock() bool {
	s.logger.Debugln("sendRequests()")
	var sent bool

	for _, pack := range s.getPacks() {
		peer := s.readyPool.pop()
		s.logger.Tracef("sendRequests() peer=%v pack=%d", peer.id, len(pack))
		if err := peer.RequestData(pack, s.HandleData); err == nil {
			s.sentPool.push(peer)
			sent = true
		} else {
			s.logger.Debugf("sendRequests() failed by %+v", err)
			s.checkedPool.push(peer)
		}
	}

	return sent
}

func (s *syncProcessor) next() bool {
	if s.reqIter == nil {
		s.reqIter = s.builder.Requests()
	}
	if s.reqCount < configRoundLimit && s.reqIter.Next() {
		s.reqCount += 1
		return true
	}

	s.reqCount = 0
	s.reqIter = nil
	return false
}

func (s *syncProcessor) getPacks() [][]BucketIDAndBytes {
	peerSize := s.readyPool.size()
	if peerSize == 0 {
		s.logger.Warn("getPacks() No peers to request")
		return nil
	}

	var packs [][]BucketIDAndBytes

	pack := make([]BucketIDAndBytes, 0, configPackSize)
	for s.next() {
		reqData := BucketIDAndBytes{
			BkID:  s.reqIter.BucketIDs()[0],
			Bytes: s.reqIter.Key(),
		}
		pack = append(pack, reqData)

		if len(pack) == configPackSize {
			packs = append(packs, pack)
			pack = make([]BucketIDAndBytes, 0, configPackSize)
		}

		if len(packs) == peerSize && s.reqCount < configRoundLimit {
			break
		}
	}

	if len(pack) > 0 {
		packs = append(packs, pack)
	}

	return packs
}

func (s *syncProcessor) wakeupInLock() {
	s.waiter.Signal()
}

func (s *syncProcessor) migrate() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.checkedPool == nil {
		return
	}

	s.logger.Tracef("migrate() checkPool=%d", s.checkedPool.size())
	if s.checkedPool.size() > 0 {
		if s.readyPool.size() == 0 {
			s.readyPool, s.checkedPool = s.checkedPool, s.readyPool
		} else {
			for _, p := range s.checkedPool.peerList() {
				s.readyPool.push(p)
			}
			s.checkedPool.clear()
		}
	}

	s.timer = nil
	s.wakeupInLock()
}

func (s *syncProcessor) onPoolChangeInLock() {
	if s.readyPool.size() > 0 {
		s.wakeupInLock()
		return
	}
	if s.sentPool.size() == 0 && s.checkedPool.size() > 0 {
		if s.timer != nil {
			s.logger.Debugf("onPoolChangeInLock() timer=%v exist", s.timer)
		} else {
			s.logger.Tracef("onPoolChangeInLock() migrate timer start. duration=%v", s.migrateDur)
			s.timer = time.AfterFunc(s.migrateDur, s.migrate)
		}
	}
}

// HandleData handle data from peer. If it expires timeout, data would
// be nil.
func (s *syncProcessor) HandleData(reqID uint32, sender *peer, data []BucketIDAndBytes) {
	s.logger.Debugf("HandleData()")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.logger.Tracef("HandleData() reqID=%d sender=%v data=%d", reqID, sender.id, len(data))

	if s.builder == nil || s.sentPool == nil {
		s.logger.Tracef("HandleData() syncProcessor stopped or finished")
		return
	}

	p := s.sentPool.remove(sender.id)
	if p == nil {
		s.logger.Debugf("HandleData() peer=%v not in sentPool", sender.id)
		return
	}

	var hasError bool
	var received int
	for _, item := range data {
		if err := s.builder.OnData(item.BkID, item.Bytes); err == nil {
			received += 1
		} else {
			if err != merkle.ErrNoRequester {
				hasError = true
				s.logger.Warnf("HandleData() failed builder.OnData err=%v item=%v", err, item)
			}
		}
	}

	s.logger.Tracef("HandleData() reqID=%d data=%d received=%d hasError=%v", reqID, len(data), received, hasError)
	if len(data) > 0 && !hasError {
		s.readyPool.push(p)
	} else {
		s.checkedPool.push(p)
	}
	s.onPoolChangeInLock()
}

func newSyncProcessor(builder merkle.Builder, reactors []SyncReactor, logger log.Logger, datasyncer bool) *syncProcessor {
	sp := &syncProcessor{
		builder:     builder,
		reactors:    reactors,
		readyPool:   newPeerPool(),
		sentPool:    newPeerPool(),
		checkedPool: newPeerPool(),
		datasyncer:  datasyncer,
	}
	sp.waiter = sync.NewCond(&sp.mutex)

	sp.logger = logger.WithFields(log.Fields{
		log.FieldKeyPrefix: fmt.Sprintf("SyncProcessor[%p] ", sp),
	})
	return sp
}
