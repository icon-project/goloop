package sync2

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
)

const (
	configChannelSize int = 10
	configPackSize    int = 10
	configRoundLimit  int = 500

	WakeUp string = "WAKE_UP"
	Done   string = "DONE"
)

type SyncProcessor interface {
	Start(cb func(err error))
	Stop()
	AddRequest(id db.BucketID, key []byte) error
}

type syncProcessor struct {
	mutex      sync.Mutex
	logger     log.Logger
	notifyDone sync.Once

	builder  merkle.Builder
	reactors []SyncReactor

	// reactor ready+sent+checked
	// ready --> sent -(no valid)-> checked -(no ready/sent) -> ready
	//                -(valid)----> ready
	readyPool   *peerPool
	sentPool    *peerPool
	checkedPool *peerPool

	msgCh      chan interface{}
	timer      *time.Timer
	awaking    bool
	datasyncer bool

	reqIter  merkle.RequestIterator
	reqCount int
}

func (s *syncProcessor) cleanup() {
	s.logger.Debugln("cleanup")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.stopTimerInLock()
	for _, p := range s.sentPool.peerList() {
		p.Reset()
	}

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
		s.prepareWakeupInLock()
		return
	}
	if p2 := s.sentPool.remove(p.id); p2 != nil {
		s.prepareWakeupInLock()
		return
	}
	if p2 := s.checkedPool.remove(p.id); p2 != nil {
		return
	}
}

func (s *syncProcessor) initReadyPool() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

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

	err = s.doSync()
}

func (s *syncProcessor) stopTimer() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.awaking {
		s.awaking = false
	}

	s.stopTimerInLock()
}

func (s *syncProcessor) stopTimerInLock() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *syncProcessor) processMessage(msg interface{}) (bool, error) {
	switch msgType := msg.(type) {
	case string:
		switch msgType {
		case Done:
			if !s.datasyncer {
				s.logger.Infof("processMessage() done sync processor")
				return true, nil
			}
		case WakeUp:
			s.stopTimer()
			s.sendRequests()
		}
	case error:
		err := msgType
		switch err {
		case errors.ErrInterrupted:
			s.stopTimer()
			s.logger.Infof("processMessage() stop sync processor by %v", err)
			return true, err
		default:
			s.logger.Panicf("processMessage() undefined err(%v)\n", err)
		}
	default:
		s.logger.Warnf("processMessage() unknown type(%v)\n", msgType)
	}

	return false, nil
}

func (s *syncProcessor) doSync() error {
	defer s.cleanup()

	var msg interface{}
	s.initReadyPool()

	for {
		s.mutex.Lock()
		count := s.builder.UnresolvedCount()
		s.logger.Tracef("doSync() unresolvedCount(%d)", count)
		peerSize := s.readyPool.size()
		s.mutex.Unlock()

		if (s.datasyncer && count == 0) || peerSize == 0 || len(s.msgCh) > 0 {
			s.logger.Tracef("doSync() waiting message from channel... count(%d), peerSize(%d), msgChLen(%d)", count, peerSize, len(s.msgCh))
			msg = <-s.msgCh
		} else if count > 0 {
			msg = WakeUp
		} else {
			msg = nil
		}
		s.logger.Tracef("doSync() peerSize(%v), msgCh(%d), msg(%v)", peerSize, len(s.msgCh), msg)

		if done, err := s.processMessage(msg); done {
			return err
		}
	}
}

// stop sync processor
func (s *syncProcessor) Stop() {
	s.logger.Debugln("Stop() sync processor")
	s.msgCh <- errors.ErrInterrupted
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
		s.logger.Infof("AddRequest() REQUEST id=%s key=%#x", id, key)
		s.builder.RequestData(id, key, onDataHandler(func() {
			s.logger.Infof("AddRequest() ADD id=%s key=%#x", id, key)
		}))

		if !s.awaking {
			s.awaking = true
			s.msgCh <- WakeUp
		}
		return err
	} else {
		return errors.InvalidStateError.Errorf("Terminated")
	}
}

// syncProcessor --> peer --> PeerHandler(Reactor) --> module.ProtocolHandler
func (s *syncProcessor) sendRequests() {
	s.logger.Debugln("sendRequests()")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, pack := range s.getPacks() {
		peer := s.readyPool.pop()
		if err := peer.RequestData(pack, s.HandleData); err == nil {
			s.sentPool.push(peer)
		} else {
			s.logger.Infof("Request failed by %+v", err)
			s.checkedPool.push(peer)
		}
	}
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

func (s *syncProcessor) wakeup() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.timer == nil {
		s.logger.Infof("wakeup() timer(%v) already cancelled", s.timer)
		return
	}

	readyPoolSize := s.readyPool.size()
	sentPoolSize := s.sentPool.size()

	if s.checkedPool.size() > 0 {
		if (readyPoolSize + sentPoolSize) == 0 {
			s.readyPool, s.checkedPool = s.checkedPool, s.readyPool
		} else if sentPoolSize == 0 { // no request to receive data from peer
			for _, p := range s.checkedPool.peerList() {
				s.readyPool.push(p)
			}
			s.checkedPool.clear()
		}
	}

	s.stopTimerInLock()

	s.msgCh <- WakeUp
}

func (s *syncProcessor) prepareWakeupInLock() {
	if s.timer != nil {
		s.logger.Infof("prepareWakeUpInLock() timer already started")
		return
	}

	if s.awaking {
		s.logger.Infof("prepareWakeUpInLock() awaking...")
		return
	}

	if len(s.msgCh) == configChannelSize {
		s.logger.Panicf("prepareWakeUpInLock() message channel is full"+
			" len(msgCh) :%v, cap(msgCh) :%v", len(s.msgCh), cap(s.msgCh))
		return
	}

	var timeInterval = configDiscoveryInterval
	if s.readyPool.size() > 0 && s.sentPool.size() == 0 {
		s.logger.Errorf("prepareWakeUpInLock() sentPool(%d)", s.sentPool.size())
		timeInterval = 0
	}

	s.awaking = true
	s.logger.Infof("prepareWakeUpInLock() wakeup after %f seconds", (timeInterval.Seconds()))
	s.timer = afterFunc(timeInterval, s.wakeup)
}

// HandleData handle data from peer. If it expires timeout, data would
// be nil.
func (s *syncProcessor) HandleData(reqID uint32, sender *peer, data []BucketIDAndBytes) {
	s.logger.Tracef("HandleData() reqID=%d sender=%v", reqID, sender.id)
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.sentPool == nil {
		s.logger.Warnf("HandleData() sendPool is %v", s.sentPool)
		return
	}

	p := s.sentPool.remove(sender.id)
	if p == nil {
		s.logger.Warnf("HandleData() peer(%v) not in sentPool", sender.id)
		return
	}

	var received int
	for _, item := range data {
		if err := s.builder.OnData(item.BkID, item.Bytes); err == nil {
			received += 1
		} else {
			s.logger.Errorf("HandleData() failed builder.OnData err(%v)", err)
		}
	}

	count := s.builder.UnresolvedCount()
	if count == 0 {
		s.notifyDone.Do(func() {
			s.logger.Infof("HandleData() notify Done")
			s.msgCh <- Done
		})
		return
	}

	s.logger.Tracef("HandleData() received : %d", received)
	if received > 0 {
		s.readyPool.push(p)
	} else {
		s.checkedPool.push(p)
	}

	s.prepareWakeupInLock()
}

func newSyncProcessor(builder merkle.Builder, reactors []SyncReactor, log log.Logger, datasyncer bool) *syncProcessor {
	return &syncProcessor{
		logger:      log,
		builder:     builder,
		reactors:    reactors,
		readyPool:   newPeerPool(),
		sentPool:    newPeerPool(),
		checkedPool: newPeerPool(),
		msgCh:       make(chan interface{}, configChannelSize),
		datasyncer:  datasyncer,
	}
}
