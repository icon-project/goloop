package sync

import (
	"sync"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"github.com/icon-project/goloop/service/txresult"
)

/*
pool management
pool - all peers connected are in this
vpool - push peer to vpool after peer is removed from sentReq when status of OnResult is NoError
ivpool
sentReq - push peer to sentReq after call client.XXX and remove the peer from the pool which the peer was in
*/

type syncType int

const (
	syncWorldState syncType = 1 << iota
	syncPatchReceipts
	syncNormalReceipts
	end
)

func (s syncType) toIndex() int {
	var index int
	switch s {
	case syncWorldState:
		index = 0
	case syncPatchReceipts:
		index = 1
	case syncNormalReceipts:
		index = 2
	}
	return index
}

func (s syncType) String() string {
	var str string
	switch s {
	case syncWorldState:
		str = "syncWorldState"
	case syncPatchReceipts:
		str = "syncPatchReceipts"
	case syncNormalReceipts:
		str = "syncNormalReceipts"
	}
	return str
}

func (s syncType) isValid() bool {
	return s < end
}

const (
	syncComplete         = int(syncWorldState | syncPatchReceipts | syncNormalReceipts)
	configMaxPeerForSync = 5
)

type syncer struct {
	// mutex is used for vpool, ivpool and sentReq
	mutex sync.Mutex
	cond  *sync.Cond

	client   *client
	database db.Database

	pool    *peerPool
	vpool   *peerPool
	ivpool  *peerPool
	sentReq map[module.PeerID]*peer
	reqKey  map[string]bool

	builder [3]merkle.Builder

	ah  []byte
	vlh []byte
	prh []byte
	nrh []byte

	finishCh chan syncType
	log      log.Logger

	wss      state.WorldSnapshot
	prl      module.ReceiptList
	nrl      module.ReceiptList
	complete int
	cb       func(syncing bool)
}

type Request struct {
	reqID uint32
	pi    module.ProtocolInfo
}

type Callback interface {
	onResult(status errCode, p *peer)
	onNodeData(p *peer, status errCode, t syncType, data [][]byte)
}

func (s *syncer) onResult(status errCode, p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Debugf("OnResult : status(%d), p(%s)\n", status, p)
	if status == NoError {
		s.vpool.push(p.id, p)
		if s.vpool.size() == 1 {
			s.cond.Signal()
		}
	} else {
		s._requestIfNotEnough(p)
	}
}

func (s *syncer) onNodeData(p *peer, status errCode, t syncType, data [][]byte) {
	s.log.Debugf("OnNodeData p(%s), status(%d), t(%d), data(%#x)\n", p, status, t, data)
	s.mutex.Lock()
	s.vpool.push(p.id, p)
	if s.vpool.size() == 1 {
		s.cond.Signal()
	}

	if t.isValid() == false {
		s.log.Warnf("Wrong syncType. (%d)\n", t)
		s.mutex.Unlock()
		return
	}

	builder := s.builder[t.toIndex()]
	if status != NoError {
		s._requestIfNotEnough(p)
		s.mutex.Unlock()
	} else {
		s.mutex.Unlock()
		for _, d := range data {
			s.log.Debugf("receive node(%#x)\n", d)
			if err := builder.OnData(d); err != nil {
				s.log.Infof("Failed to OnData to builder data(%#x), err(%+v)\n", d, err)
			}
		}
	}

	if s._reqUnresolvedNode(t, builder) == 0 {
		s.mutex.Lock()
		s.finishCh <- t
		s.mutex.Unlock()
	}
}

func (s *syncer) onReceive(pi module.ProtocolInfo, b []byte, p *peer) (bool, error) {
	s.log.Debugf("syncer onReceive pi(%s), b(%#x), p(%s)\n", pi, b, p)
	s.processMsg(receiveMsg, pi, b, p)
	return false, nil
}

func (s *syncer) onJoin(p *peer) {
	log.Debugf("syncer.onJoin peer(%s)\n", p)
	p.cb = s
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s._requestIfNotEnough(p)
}

func (s *syncer) onLeave(id module.PeerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	p := s.vpool.getPeer(id)
	if p == nil {
		// TODO remove rp then remove cancleCh do stop(rp.timer)
		if rp := s.sentReq[id]; rp != nil {
			rp.timer.Stop()
			delete(s.sentReq, id)
		}
		return
	}
	s.vpool.remove(id)
}

func (s *syncer) processMsg(msgType int, pi module.ProtocolInfo, b []byte, p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	rp := s.sentReq[p.id]
	if rp == nil {
		// already consumed request
		return
	}
	var i interface{}
	var reqID uint32
	var err error
	switch pi {
	case protoResult:
		data := new(result)
		_, err = c.UnmarshalFromBytes(b, data)
		reqID = data.ReqID
		i = data
	case protoNodeData:
		data := new(nodeData)
		_, err = c.UnmarshalFromBytes(b, data)
		reqID = data.ReqID
		i = data
	default:
		log.Info("Invalid protocol received(%d)\n", pi)
	}

	if err != nil || reqID != p.reqID {
		s.log.Infof("Failed onReceive. err(%v), receivedReqID(%d), p.reqID(%d), pi(%s)\n", err, reqID, p.reqID, pi)
		return
	}
	delete(s.sentReq, p.id)

	go p.onReceive(msgType, pi, i)
}

func (s *syncer) stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for k, p := range s.sentReq {
		delete(s.sentReq, k)
		p.timer.Stop()
		s.vpool.push(p.id, p)
	}
	s.cond.Broadcast()
	close(s.finishCh)
}

func (s *syncer) _updateValidPool() {
	// TODO increase configMaxPeerForSync
	// This call means validpool is not enough
	if s.ivpool.size() == 0 {
		return
	}

	var failedList []*peer
	for p := s.ivpool.pop(); p != nil; {
		err := s.client.hasNode(
			p, s.ah, s.prh,
			s.nrh, s.vlh)
		if err != nil {
			failedList = append(failedList, p)
		}
		s.sentReq[p.id] = p
	}

	for _, p := range failedList {
		s.ivpool.push(p.id, p)
	}
}

func (s *syncer) _requestIfNotEnough(p *peer) {
	log.Debugf("vpool(%d), pool(%d)\n", s.vpool.size(), s.pool.size())
	if s.vpool.size() < configMaxPeerForSync && s.vpool.size() != s.pool.size() {
		go func() {
			s.mutex.Lock()
			defer s.mutex.Unlock()
			err := s.client.hasNode(
				p, s.ah, s.prh,
				s.nrh, s.vlh)
			if err != nil {
				log.Info("Failed to request hasNode to %s, err(%+v)\n", p, err)
				s.ivpool.push(p.id, p)
			} else {
				s.sentReq[p.id] = p
			}
		}()
	} else {
		s.ivpool.push(p.id, p)
	}
}

func (s *syncer) _getValidPeer() *peer {
	var p *peer
	for p = s.vpool.pop(); p == nil; p = s.vpool.pop() {
		s._updateValidPool()
		s.log.Debug("_reqUnresolvedNode waiting for valid peer\n")
		s.cond.Wait()
		s.log.Debug("_reqUnresolvedNode wake up \n")
	}
	return p
}

func (s *syncer) _reqUnresolvedNode(t syncType, builder merkle.Builder) int {
	unresolved := builder.UnresolvedCount()
	s.log.Debugf("_reqUnresolvedNode unresolved(%d)\n", unresolved)
	if unresolved == 0 {
		s.mutex.Lock()
		s.finishCh <- t
		s.mutex.Unlock()
		return 0
	}
	req := builder.Requests()

	var keys [][]byte
	s.mutex.Lock()
	for req.Next() {
		key := req.Key()
		s.log.Debugf("request node(%#x)\n", key)
		keys = append(keys, key)
		s.reqKey[string(key)] = true
	}
	p := s._getValidPeer()
	s.mutex.Unlock()
	if err := s.client.requestNodeData(p, keys, t, s.processMsg); err != nil {
		// TODO request node data with another
	}
	s.mutex.Lock()
	s.sentReq[p.id] = p
	s.mutex.Unlock()
	return unresolved
}

func (s *syncer) start() {
	s.cb(true)
	s.mutex.Lock()
	defer s.mutex.Unlock()

	pl := s.pool.peerList()
	for _, p := range pl {
		p.cb = s
		err := s.client.hasNode(p, s.ah, s.prh, s.nrh, s.vlh)
		if err != nil {
			s.log.Info("Failed to request hasNode to %s, err(%+v)\n", p, err)
		}
		s.sentReq[p.id] = p
	}

	f := func(t syncType, builder merkle.Builder) {
		s.log.Debugf("start sync for (%s)\n", t)
		if s._reqUnresolvedNode(t, builder) == 0 {
			s.finishCh <- t
		}
	}

	builder := merkle.NewBuilder(s.database)
	s.builder[syncWorldState.toIndex()] = builder
	if wss, err := state.NewWorldSnapshotWithBuilder(builder, s.ah, s.vlh); err == nil {
		s.wss = wss
	} else {
		s.log.Panicf("Failed to call NewWorldSnapshotWithBuilder, ah(%#x), vlh(%#x)\n", s.ah, s.vlh)
	}
	go f(syncWorldState, builder)

	rf := func(t syncType, rl *module.ReceiptList) {
		if s.nrh != nil {
			builder := merkle.NewBuilder(s.database)
			s.builder[t.toIndex()] = builder
			*rl = txresult.NewReceiptListWithBuilder(builder, s.nrh)
			go f(t, builder)
		} else {
			s.nrl = txresult.NewReceiptListFromSlice(s.database, []txresult.Receipt{})
			s.complete |= int(t)
		}
	}

	rf(syncPatchReceipts, &s.prl)
	rf(syncNormalReceipts, &s.nrl)
}

func (s *syncer) done() *Result {
	end := s.complete
	for end != syncComplete {
		c := <-s.finishCh
		end |= int(c)
		s.log.Infof("done sync (%d) / (%d)\n", c, end)
	}
	s.cb(false)
	return &Result{s.wss, s.prl, s.nrl}
}

func (s *syncer) ForceSync() *Result {
	s.start()
	r := s.done()
	return r
}

func (s *syncer) Finalize() error {
	s.log.Debugf("Finalize :  ah(%#x), patchHash(%#x), normalHash(%#x), vlh(%#x)\n",
		s.ah, s.prh, s.nrh, s.vlh)
	//for i, builder := range []merkle.Builder{s.wsBuilder, s.prBuilder, s.nrBuilder} {
	for i, t := range []syncType{syncWorldState, syncPatchReceipts, syncNormalReceipts} {
		builder := s.builder[t.toIndex()]
		if builder == nil {
			continue
		} else {
			if err := builder.Flush(true); err != nil {
				s.log.Errorf("Failed to flush for %d builder err(%+v)\n", i, err)
				return err
			}
		}
	}
	return nil
}

func newSyncer(database db.Database, c *client, p *peerPool,
	accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash []byte, log log.Logger, cb func(syncing bool)) *syncer {
	log.Debugf("newSyncer ah(%#x), pReceiptsHash(%#x), nReceiptsHash(%#x), vlh(%#x)\n", accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash)

	s := &syncer{
		database: database,
		pool:     p,
		client:   c,
		vpool:    newPeerPool(),
		ivpool:   newPeerPool(),
		sentReq:  make(map[module.PeerID]*peer),
		ah:       accountsHash,
		prh:      pReceiptsHash,
		nrh:      nReceiptsHash,
		vlh:      validatorListHash,
		reqKey:   make(map[string]bool),
		finishCh: make(chan syncType),
		log:      log,
		cb:       cb,
	}
	s.cond = sync.NewCond(&s.mutex)
	return s
}
