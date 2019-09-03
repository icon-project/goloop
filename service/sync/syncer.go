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

type syncType int

const (
	syncWorldState syncType = 1 << iota
	syncPatchReceipts
	syncNormalReceipts
	end
)

const (
	configMaxRequestHash = 20
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
	configMaxPeerForSync = 10
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

	builder [3]merkle.Builder
	bMutex  [3]sync.Mutex

	ah  []byte
	vlh []byte
	prh []byte
	nrh []byte

	finishCh chan syncType
	log      log.Logger

	wss state.WorldSnapshot
	prl module.ReceiptList
	nrl module.ReceiptList
	cb  func(syncing bool)

	waitingPeerCnt int
	complete       int
}

type Request struct {
	reqID uint32
	pi    module.ProtocolInfo
}

type Callback interface {
	onResult(status errCode, p *peer)
	onNodeData(p *peer, status errCode, t syncType, data [][]byte)
	onReceive(pi module.ProtocolInfo, b []byte, p *peer)
}

func (s *syncer) _reqUnresolvedNode(st syncType, builder merkle.Builder,
	peers []*peer) (bool, int, []*peer) {
	unresolved := builder.UnresolvedCount()
	s.log.Debugf("_reqUnresolvedNode unresolved(%d) for (%s) len(peers) = %d\n",
		unresolved, st, len(peers))
	if unresolved == 0 {
		return true, 0, peers
	}
	req := builder.Requests()

	keyMap := make(map[string]bool)
	var keys [][]byte
	reqNum := 0
	for req.Next() {
		key := req.Key()
		if keyMap[string(key)] == false {
			keyMap[string(key)] = true
			keys = append(keys, key)
			reqNum++
		}
	}

	need := reqNum/configMaxRequestHash + 1
	var unusedPeers []*peer
	peerNum := len(peers)

	if need > peerNum {
		need = peerNum
	}

	result := false
	i := 0
	pIndex := 0
	for j, p := range peers {
		offset := (reqNum * i) / need
		end := (reqNum * (i + 1)) / need
		pIndex = j
		s.log.Debugf("requestNodeData for (%d) -> (%d) reqNum(%d)\n", offset, end, reqNum)
		if err := s.client.requestNodeData(p, keys[offset:end], st, s.processMsg); err != nil {
			s.log.Debugf("Failed to request node\n")
			unusedPeers = append(unusedPeers, p)
			continue
		}
		i++
		if result == false {
			result = true
		}
		if need == i {
			break
		}
	}
	for ; pIndex+1 < peerNum; pIndex++ {
		unusedPeers = append(unusedPeers, peers[pIndex+1])
	}
	// return unused peers
	s.log.Debugf("requestNodeData unusedPeers(%d)\n", len(unusedPeers))
	return result, 1, unusedPeers
}

func (s *syncer) _onNodeData(builder merkle.Builder, data [][]byte) int {
	if len(data) != 0 {
		s.log.Debugf("Received len(data) (%d)\n", len(data))
	}
	for _, d := range data {
		s.log.Debugf("receive node(%#x)\n", d)
		if err := builder.OnData(d); err != nil {
			s.log.Infof("Failed to OnData to builder data(%#x), err(%+v)\n", d, err)
		}
	}
	return builder.UnresolvedCount()
}

func (s *syncer) reqUnresolved(st syncType, builder merkle.Builder, need int) {
	f := func(need int) []*peer {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		var peers []*peer
		for {
			s.log.Debugf("Wait for valid peer st(%s)\n", st)
			peers = s._getValidPeers(need)
			if peers != nil {
				break
			}
			s.waitingPeerCnt++
			s.cond.Wait()
			s.waitingPeerCnt--
			if s.complete&int(st) == int(st) {
				return nil
			}
			s.log.Debugf("Wake up for valid peer st(%s)\n", st)
		}
		return peers
	}

	mutex := &s.bMutex[st.toIndex()]
	for {
		peers := f(need)
		if len(peers) == 0 {
			return
		}
		mutex.Lock()
		b, unresolved, unused := s._reqUnresolvedNode(st, builder, peers)
		mutex.Unlock()
		if b == false {
			s.mutex.Lock()
			s._returnValidPeers(unused)
			s.mutex.Unlock()
			continue
		} else {
			s.mutex.Lock()
			size := s.vpool.size()
			if len(unused) > 0 {
				s._returnValidPeers(unused)
			}
			s.log.Debugf("st(%s), size(%d), unused(%d), unresolved(%d)\n",
				st, size, len(unused), unresolved)
			if unresolved == 0 {
				if s.complete&int(st) != int(st) {
					s.finishCh <- st
				}
			}
			if size == 0 && len(unused) > 0 && s.complete != syncComplete {
				s.cond.Signal()
			}
			s.mutex.Unlock()
			break
		}
	}
}

func (s *syncer) onNodeData(p *peer, status errCode, st syncType, data [][]byte) {
	s.mutex.Lock()
	s.vpool.push(p)
	if s.waitingPeerCnt > 0 {
		s.cond.Signal()
	}
	s.mutex.Unlock()

	if st.isValid() == false {
		s.log.Warnf("Wrong syncType. (%d)\n", st)
		return
	}

	bIndex := st.toIndex()
	builder := s.builder[bIndex]

	s.bMutex[bIndex].Lock()
	unresolved := s._onNodeData(builder, data)
	s.bMutex[bIndex].Unlock()

	if unresolved == 0 {
		s.mutex.Lock()
		if s.complete&int(st) != int(st) {
			s.complete |= int(st)
			s.finishCh <- st
		}
		s.mutex.Unlock()

	}
	need := unresolved/configMaxRequestHash + 1
	s.reqUnresolved(st, builder, need)
}

func (s *syncer) onReceive(pi module.ProtocolInfo, b []byte, p *peer) {
	s.processMsg(receiveMsg, pi, b, p)
}

func (s *syncer) onJoin(p *peer) {
	log.Debugf("onJoin peer(%s)\n", p)
	p.cb = s
	s.mutex.Lock()
	s._requestIfNotEnough(p)
	s.mutex.Unlock()
}

func (s *syncer) onLeave(id module.PeerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Debugf("onLeave id(%s)\n", id)
	p := s.vpool.getPeer(id)
	if p == nil {
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
		log.Debugf("peer(%s) for (%s) is already received\n", p, pi)
		return
	}

	var i interface{}
	if msgType == receiveMsg {
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
			return
		}

		if err != nil || reqID != p.reqID {
			s.log.Infof(
				"Failed onReceive. err(%v), receivedReqID(%d), p.reqID(%d), pi(%s), byte(%#x)\n",
				err, reqID, p.reqID, pi, b)
			return
		}
	}
	p.timer.Stop()
	delete(s.sentReq, p.id)

	go func() {
		if p.onReceive(msgType, pi, i) == false {
			s.vpool.push(p)
		}
	}()
}

func (s *syncer) stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for k, p := range s.sentReq {
		delete(s.sentReq, k)
		p.timer.Stop()
		s.vpool.push(p)
	}
	s.cond.Broadcast()
	close(s.finishCh)
}

func (s *syncer) _updateValidPool() {
	if s.ivpool.size() == 0 {
		return
	}

	var failedList []*peer
	for p := s.ivpool.pop(); p != nil; p = s.ivpool.pop() {
		err := s.client.hasNode(
			p, s.ah, s.prh,
			s.nrh, s.vlh, s.processMsg)
		if err != nil {
			failedList = append(failedList, p)
		}
		s.sentReq[p.id] = p
	}

	for _, p := range failedList {
		s.ivpool.push(p)
	}
}

func (s *syncer) _requestIfNotEnough(p *peer) {
	log.Debugf("vpool(%d), pool(%d)\n", s.vpool.size(), s.pool.size())
	if s.vpool.size() < configMaxPeerForSync && s.vpool.size() != s.pool.size() {
		err := s.client.hasNode(
			p, s.ah, s.prh,
			s.nrh, s.vlh, s.processMsg)
		if err != nil {
			log.Info("Failed to request hasNode to %s, err(%+v)\n", p, err)
			s.ivpool.push(p)
		} else {
			s.sentReq[p.id] = p
		}
	} else {
		s.ivpool.push(p)
	}
}

func (s *syncer) _returnValidPeers(peers []*peer) {
	for _, peer := range peers {
		delete(s.sentReq, peer.id)
		s.vpool.push(peer)
	}

	if s.waitingPeerCnt > 0 {
		s.cond.Signal()
	}
}

func (s *syncer) _getValidPeers(need int) []*peer {
	if s.vpool.size() == 0 {
		go s._updateValidPool()
		s.log.Debugf("_getValidPeers size = %d\n", s.vpool.size())
		return nil
	}
	size := s.vpool.size()
	if size > need {
		size = need
	}
	peers := make([]*peer, size)
	for i := 0; i < size; i++ {
		peer := s.vpool.pop()
		s.sentReq[peer.id] = peer
		peers[i] = peer
	}
	s.log.Debugf("_getValidPeers size = %d, peers = %v\n", size, peers)
	return peers
}

func (s *syncer) onResult(status errCode, p *peer) {
	log.Debugf("OnResult : status(%d), p(%s)\n", status, p)
	if status == NoError {
		s.mutex.Lock()
		s.vpool.push(p)
		if s.waitingPeerCnt > 0 {
			s.cond.Signal()
		}
		s.mutex.Unlock()
	} else {
		s.mutex.Lock()
		s._requestIfNotEnough(p)
		s.mutex.Unlock()
	}
}

func (s *syncer) ForceSync() *Result {
	s.log.Debugf("ForceSync")
	s.cb(true)

	pl := s.pool.peerList()
	s.mutex.Lock()
	for _, p := range pl {
		p.cb = s
		err := s.client.hasNode(p, s.ah, s.prh, s.nrh, s.vlh, s.processMsg)
		if err != nil {
			s.log.Info("Failed to request hasNode to %s, err(%+v)\n", p, err)
		}
		s.sentReq[p.id] = p
	}
	s.mutex.Unlock()

	builder := merkle.NewBuilder(s.database)
	s.builder[syncWorldState.toIndex()] = builder
	if wss, err := state.NewWorldSnapshotWithBuilder(builder, s.ah, s.vlh); err == nil {
		s.wss = wss
	} else {
		s.log.Panicf("Failed to call NewWorldSnapshotWithBuilder, ah(%#x), vlh(%#x)\n", s.ah, s.vlh)
	}
	go s.reqUnresolved(syncWorldState, builder, 1)

	rf := func(t syncType, rl *module.ReceiptList, rh []byte) {
		if len(rh) != 0 {
			builder := merkle.NewBuilder(s.database)
			s.builder[t.toIndex()] = builder
			*rl = txresult.NewReceiptListWithBuilder(builder, rh)
			go s.reqUnresolved(t, builder, 1)
		} else {
			*rl = txresult.NewReceiptListFromSlice(s.database, []txresult.Receipt{})
			s.mutex.Lock()
			s.complete |= int(t)
			s.mutex.Unlock()
		}
	}

	rf(syncPatchReceipts, &s.prl, s.prh)
	rf(syncNormalReceipts, &s.nrl, s.nrh)

	for s.complete != syncComplete {
		c := <-s.finishCh
		s.mutex.Lock()
		s.complete |= int(c)
		s.log.Debugf("complete (%b) / (%b)\n", c, s.complete)
		s.mutex.Unlock()
	}
	s.cb(false)
	return &Result{s.wss, s.prl, s.nrl}
}

func (s *syncer) Finalize() error {
	s.log.Debugf("Finalize :  ah(%#x), patchHash(%#x), normalHash(%#x), vlh(%#x)\n",
		s.ah, s.prh, s.nrh, s.vlh)
	for i, t := range []syncType{syncWorldState, syncPatchReceipts, syncNormalReceipts} {
		builder := s.builder[t.toIndex()]
		if builder == nil {
			continue
		} else {
			s.log.Debugf("Flush %s\n", t)
			if err := builder.Flush(true); err != nil {
				s.log.Errorf("Failed to flush for %d builder err(%+v)\n", i, err)
				return err
			}
		}
	}
	return nil
}

func newSyncer(database db.Database, c *client, p *peerPool,
	accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash []byte,
	log log.Logger, cb func(syncing bool)) *syncer {
	log.Debugf("newSyncer ah(%#x), pReceiptsHash(%#x), nReceiptsHash(%#x), vlh(%#x)\n",
		accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash)

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
		finishCh: make(chan syncType, 1),
		log:      log,
		cb:       cb,
	}
	s.cond = sync.NewCond(&s.mutex)
	return s
}
