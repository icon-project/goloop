package sync2

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
)

const packetSize = 10

type syncState int

const (
	InitState syncState = iota
	SentState
	HasState
	NoDataState
	DoneState
)

var (
	ErrNoPeers       = errors.New("no peers")
	ErrUpdatedPeers  = errors.New("updated peers")
	ErrEmptyResponse = errors.New("empty response")
)

type SyncProcessor interface {
	StartSync() error
	Stop()
	GetBuilder() merkle.Builder
	AddRequest(id db.BucketID, key []byte) error
}

type syncProcessor struct {
	mutex sync.Mutex
	log   log.Logger
	state syncState

	builder    merkle.Builder
	reactors   []SyncReactor
	spCh       chan error    // channel for syncer processor
	notiCh     chan struct{} // notification channel for peer updated
	dsCh       chan bool     // channel for data syncer processor
	stopCh     chan bool     // channel for sync processor stop
	ticker     *time.Ticker
	tickerDone chan bool
	datasyncer bool

	// reactor ready+sent+checked
	// ready --> sent -(no valid)-> checked -(no ready/sent) -> ready
	//                -(valid)----> ready
	readyPool   *peerPool
	sentPool    *peerPool
	checkedPool *peerPool
}

// Waiting for requests to be added when data syncer
func (s *syncProcessor) waitBuilderRequest() error {
	s.log.Debugln("waitBuilderRequest() waiting for requests...")
	select {
	case requestOk := <-s.dsCh:
		s.mutex.Lock()
		count := s.builder.UnresolvedCount()
		s.mutex.Unlock()
		if !requestOk || count == 0 {
			s.log.Warnf("waitRequest() no request added")
			return errors.ErrInvalidState
		}
	case <-s.stopCh:
		s.mutex.Lock()
		count := s.builder.UnresolvedCount()
		s.mutex.Unlock()
		s.updateState(DoneState)
		return s.onDone(count, errors.ErrInterrupted)
	}

	return nil
}

// on Init State
func (s *syncProcessor) init() error {
	s.log.Debugln("init state")

	if s.datasyncer {
		if err := s.waitBuilderRequest(); err != nil {
			return err
		}
	} else {
		count := s.builder.UnresolvedCount()
		if count == 0 {
			s.updateState(DoneState)
			return s.onDone(count, nil)
		}
	}

	if err := s.sendRequests(); err != nil {
		return err
	}

	// transition to SentState
	s.updateState(SentState)
	return s.onSent()
}

// on Sent State
func (s *syncProcessor) onSent() error {
	s.log.Debugln("onSent() sentRequest state")

	var err error

	for {
		select {
		case <-s.notiCh:
			s.log.Debugf("onSent() received notification that updated peer")
			return ErrUpdatedPeers
		default:
			// nothing to do
		}

		// waiting response
		s.log.Debugln("onSent() waiting response... ")
		select {
		case err = <-s.spCh:
			s.log.Debugf("onSent() response err(%v)", err)
		case <-s.stopCh:
			err = errors.ErrInterrupted
		}

		switch err {
		case errors.ErrInterrupted:
			s.updateState(DoneState)
			return s.onDone(s.builder.UnresolvedCount(), err)
		case ErrEmptyResponse:
			if s.getPeerSize() == 0 {
				// transition to NoDataState
				s.updateState(NoDataState)
				return s.onNoData()
			}
			continue
		case ErrNoPeers:
			return err
		}

		count := s.builder.UnresolvedCount()
		s.log.Debugf("onSent() unresolvedCount(%d)", count)
		if count > 0 {
			// transition to HasState
			s.updateState(HasState)
			if err := s.onData(); err != nil {
				return err
			}
		} else {
			if s.datasyncer {
				return nil
			}
			s.updateState(DoneState)
			return s.onDone(count, nil)
		}
	}
}

// on Has State
func (s *syncProcessor) onData() error {
	s.log.Debugln("onData state")

	err := s.sendRequests()

	// transition to SentState
	s.updateState(SentState)
	return err
}

// on NoData State : transition to InitState after swap peerpool
func (s *syncProcessor) onNoData() error {
	s.log.Debugln("onNoData state")

	s.mutex.Lock()
	s.readyPool, s.checkedPool = s.checkedPool, s.readyPool
	s.mutex.Unlock()

	return ErrNoPeers
}

// on Done State
func (s *syncProcessor) onDone(count int, err error) error {
	s.log.Debugln("onDone state")

	s.ticker.Stop()
	s.tickerDone <- true

	s.log.Debugf("onDone() unresolved count : %d\n", count)
	if count == 0 {
		s.log.Infoln("Finished by no more request data")
	} else {
		s.log.Infof("Finished by error(%v)", err)
	}

	return err
}

func (s *syncProcessor) updateState(state syncState) {
	s.state = state
}

func (s *syncProcessor) getPeerSize() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.readyPool.size() + s.sentPool.size()
}

func (s *syncProcessor) checkPeers() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var chMsg error

	if (s.readyPool.size() + s.sentPool.size()) == 0 {
		s.readyPool, s.checkedPool = s.checkedPool, s.readyPool
		chMsg = ErrNoPeers
	} else {
		if s.checkedPool.size() > 0 {
			for _, p := range s.checkedPool.peerList() {
				s.readyPool.push(p)
			}
			s.checkedPool.clear()
			chMsg = ErrUpdatedPeers
		}
	}

	return chMsg
}

func (s *syncProcessor) observePeers() {
	for {
		select {
		case <-s.tickerDone:
			return
		case <-s.ticker.C:
			chMsg := s.checkPeers()
			switch chMsg {
			case ErrNoPeers:
				s.spCh <- chMsg
			case ErrUpdatedPeers:
				s.notiCh <- struct{}{}
			}
		}
	}
}

func (s *syncProcessor) cleanup() {
	s.log.Debugln("cleanup")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.readyPool = nil
	s.sentPool = nil
	s.checkedPool = nil
}

func (s *syncProcessor) OnPeerJoin(p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.readyPool.push(p)
}

func (s *syncProcessor) OnPeerLeave(p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if p2 := s.readyPool.remove(p.id); p2 != nil {
		return
	}
	if p2 := s.sentPool.remove(p.id); p2 != nil {
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

// start sync procoessor
func (s *syncProcessor) StartSync() error {
	s.log.Debugln("start sync")
	defer s.cleanup()

	s.initReadyPool()

	s.ticker = time.NewTicker(configDiscoveryInterval)

	go s.observePeers()

	for {
		s.updateState(InitState)
		err := s.init()

		switch err {
		case ErrNoPeers:
			timer := time.NewTimer(configDiscoveryInterval)
			s.log.Infof("wait %f seconds for init to restart cause %v", (configDiscoveryInterval.Seconds()), err)
			<-timer.C
			continue
		case ErrUpdatedPeers:
			continue
		case errors.ErrInterrupted:
			return err
		case errors.ErrInvalidState:
			if s.datasyncer {
				continue
			}
		default:
			if !s.datasyncer {
				return nil
			}
		}
	}
}

// stop sync processor
func (s *syncProcessor) Stop() {
	s.log.Debugln("stop sync")
	s.stopCh <- true
}

func (s *syncProcessor) GetBuilder() merkle.Builder {
	return s.builder
}

func (s *syncProcessor) addRequest(id db.BucketID, key []byte) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if id.Hasher() == nil {
		return false, errors.IllegalArgumentError.Errorf("InvalidBucket(id=%q)", id)
	}
	bk, err := s.builder.Database().GetBucket(id)
	if err != nil {
		return false, err
	}
	if value, err := bk.Get(key); err != nil {
		return false, err
	} else if len(value) != 0 {
		return false, nil
	}
	s.log.Infof("DataSyncer: REQUEST id=%s key=%#x", id, key)
	s.builder.RequestData(id, key, onDataHandler(func() {
		s.log.Infof("DataSyncer: ADD id=%s key=%#x", id, key)
	}))

	return true, nil
}

func (s *syncProcessor) AddRequest(id db.BucketID, key []byte) error {
	if s.builder != nil {
		ok, err := s.addRequest(id, key)
		if ok {
			s.dsCh <- true
		}
		return err
	} else {
		return errors.InvalidStateError.Errorf("Terminated")
	}
}

func (s *syncProcessor) sendRequest(pack []BucketIDAndBytes) {
	s.log.Debugln("sendRequest()")

	peer := s.readyPool.pop()
	if err := peer.RequestData(pack, s.HandleData); err == nil {
		s.sentPool.push(peer)
	} else {
		s.log.Infof("Request failed by %v", err)
		s.readyPool.push(peer)
	}
}

// syncProcessor --> peer --> PeerHandler(Reactor) --> module.PeerHandler
func (s *syncProcessor) sendRequests() error {
	s.log.Debugln("sendRequests()")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	peers := s.readyPool.size()
	if peers == 0 {
		s.log.Warn("sendRequests() No peers")
		return ErrNoPeers
	}

	itr := s.builder.Requests()

	var sent int
	pack := make([]BucketIDAndBytes, 0, packetSize)
	for itr.Next() {
		reqData := BucketIDAndBytes{
			BkID:  itr.BucketIDs()[0],
			Bytes: itr.Key(),
		}
		pack = append(pack, reqData)
		if len(pack) == packetSize {
			s.sendRequest(pack)
			pack = make([]BucketIDAndBytes, 0, packetSize)
			sent += 1
		}

		if sent == peers {
			break
		}
	}

	if len(pack) > 0 && sent < peers {
		s.sendRequest(pack)
	}

	return nil
}

// HandleData handle data from peer. If it expires timeout, data would
// be nil.
func (s *syncProcessor) HandleData(sender *peer, data []BucketIDAndBytes) {
	s.log.Debugln("HandleData()")
	var spChMsg error

	s.mutex.Lock()
	defer func() {
		s.mutex.Unlock()
		s.spCh <- spChMsg
	}()

	p := s.sentPool.remove(sender.id)
	if p == nil {
		spChMsg = ErrEmptyResponse
		return
	}

	var received int
	for _, item := range data {
		if err := s.builder.OnData(item.BkID, item.Bytes); err == nil {
			received += 1
		} else {
			s.log.Errorf("HandleData() failed builder.OnData err(%v)\n", err)
		}
	}

	s.log.Debugln("HandleData() received :", received)
	if received > 0 {
		s.readyPool.push(p)
	} else {
		s.checkedPool.push(p)
		spChMsg = ErrEmptyResponse
	}
}

func newSyncProcessor(builder merkle.Builder, reactors []SyncReactor, log log.Logger, datasyncer bool) SyncProcessor {

	return &syncProcessor{
		log:         log,
		builder:     builder,
		reactors:    reactors,
		readyPool:   newPeerPool(),
		sentPool:    newPeerPool(),
		checkedPool: newPeerPool(),
		spCh:        make(chan error),
		notiCh:      make(chan struct{}),
		dsCh:        make(chan bool),
		stopCh:      make(chan bool),
		tickerDone:  make(chan bool),
		datasyncer:  datasyncer,
	}
}
