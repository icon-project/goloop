package sync

import (
	"fmt"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
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
	syncExtensionState
	syncTypeReserved
)

const (
	syncTypeAll          = syncTypeReserved - 1
	configMaxRequestHash = 50
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
	case syncExtensionState:
		index = 3
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
	case syncExtensionState:
		str = "syncExtensionState"
	}
	return str
}

func (s syncType) isValid() bool {
	return (s & syncTypeAll) == s
}

const (
	configMaxPeerForSync = 10
)

type syncer struct {
	mutex sync.Mutex // for peer management and complete status
	cond  *sync.Cond

	client   *client
	database db.Database
	plt      Platform
	noBuffer bool

	pool     *peerPool
	vpool    *peerPool
	ivpool   *peerPool
	sentReq  map[module.PeerID]*peer
	reqValue [4]map[string]bool

	builder  [4]merkle.Builder
	bMutex   [4]sync.Mutex // for builder
	rPeerCnt [4]int

	ah  []byte
	vlh []byte
	ed  []byte
	prh []byte
	nrh []byte

	finishCh chan error
	log      log.Logger

	wss state.WorldSnapshot
	prl module.ReceiptList
	nrl module.ReceiptList
	cb  func(syncer SyncerImpl, syncing bool)

	waitingPeerCnt int
	complete       syncType
	startTime      time.Time
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
	peers []*peer) (int, []*peer) {
	unresolved := builder.UnresolvedCount()
	s.log.Debugf("_reqUnresolvedNode unresolved(%d) for (%s) len(peers) = %d\n",
		unresolved, st, len(peers))
	if unresolved == 0 {
		return 0, peers
	}

	peerNum := 0
	if need := unresolved/configMaxRequestHash + 1; need > len(peers) {
		peerNum = len(peers)
	} else {
		peerNum = need
	}

	if mrh := peerNum * configMaxRequestHash; unresolved > mrh {
		unresolved = mrh
	}

	req := builder.Requests()
	keys := make([][]byte, 0, 100)
	reqValue := s.reqValue[st.toIndex()]
	reqNum := 0
	for reqNum < unresolved && req.Next() {
		key := req.Key()
		if b := reqValue[string(key)]; b == false {
			reqValue[string(key)] = true
			keys = append(keys, key)
			reqNum++
		}
	}

	if len(keys) == 0 && len(reqValue) != 0 {
		if s.rPeerCnt[st.toIndex()] > 0 { // another peer is running for reqValue
			s.log.Debugf("_reqUnresolvedNode keys(%d), reqValue(%d)\n", len(keys), len(reqValue))
			return unresolved, peers
		}
	}

	var unusedPeers []*peer
	i := 0
	index := 0
	for j, p := range peers {
		offset := (reqNum * i) / peerNum
		end := (reqNum * (i + 1)) / peerNum
		index = j
		s.log.Debugf("requestNodeData for %s, (%d) -> (%d) reqNum(%d)\n", st, offset, end, reqNum)
		if err := s.client.requestNodeData(p, keys[offset:end], st, s.processMsg); err != nil {
			s.log.Tracef("Failed to request node\n")
			for k := offset; k < end; k++ {
				delete(reqValue, string(keys[k]))
			}
			unusedPeers = append(unusedPeers, p)
			continue
		}
		s.rPeerCnt[st.toIndex()] += 1
		i++
		if peerNum == i {
			break
		}
	}
	unusedPeers = append(unusedPeers, peers[index+1:]...)
	s.log.Tracef("requestNodeData unusedPeers(%d)\n", len(unusedPeers))
	return unresolved, unusedPeers
}

func (s *syncer) _onNodeData(builder merkle.Builder, reqValue map[string]bool, data [][]byte, st syncType) int {
	if len(data) != 0 {
		s.log.Debugf("Received len(%d) for (%s)\n", len(data), st)
	}
	s.rPeerCnt[st.toIndex()] -= 1
	for _, d := range data {
		key := crypto.SHA3Sum256(d)
		if reqValue[string(key)] == true {
			if err := builder.OnData(d); err != nil {
				s.log.Infof("Failed to OnData to builder data(%#x), err(%+v)\n", d, err)
			}
			delete(reqValue, string(key))
		} else {
			s.log.Infof("cannot find key(%#x) in map\n", key)
		}
	}
	return builder.UnresolvedCount()
}

func (s *syncer) Complete(st syncType) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.complete&st != st {
		s.complete |= st
		if s.complete == syncTypeAll {
			s.finishCh <- nil
		}
	}
}

func (s *syncer) reqUnresolved(st syncType, builder merkle.Builder, need int) {
	mutex := &s.bMutex[st.toIndex()]
	for {
		peers := s._reservePeers(need, st)
		if len(peers) == 0 {
			s.log.Debugf("reqUnresolved peers is 0. st(%s)\n", st)
			return
		}
		mutex.Lock()
		unresolved, unused := s._reqUnresolvedNode(st, builder, peers)
		mutex.Unlock()
		s.log.Debugf("reqUnresolved Unlock << st(%s)\n", st)
		if len(unused) == len(peers) {
			s._returnPeers(unused...)
			if unresolved == 0 {
				s.Complete(st)
			}
			continue
		} else {
			if len(unused) > 0 {
				s._returnPeers(unused...)
			}
			s.log.Debugf("reqUnresolved st(%s), unused(%d), unresolved(%d)\n",
				st, len(unused), unresolved)
			if unresolved == 0 {
				s.Complete(st)
			}
			break
		}
	}
}

func (s *syncer) onNodeData(p *peer, status errCode, st syncType, data [][]byte) {
	s._returnPeers(p)
	if st.isValid() == false {
		s.log.Warnf("Wrong syncType. (%d)\n", st)
		return
	}

	if status == ErrTimeExpired {
		s.log.Debug("onNodeData TimeExpired!!\n")
		np := s._reservePeers(1, st)
		if np == nil {
			return
		}
		if err := s.client.requestNodeData(np[0], data, st, s.processMsg); err == nil {
			return
		} else {
			s._returnPeers(np...)
			s.log.Infof("Failed to request node data err(%s)\n", err)
		}
	}

	bIndex := st.toIndex()
	s.bMutex[bIndex].Lock()
	if status == ErrTimeExpired {
		for k, _ := range s.reqValue {
			delete(s.reqValue[bIndex], fmt.Sprint(k))
		}
	}
	builder := s.builder[bIndex]
	unresolved := s._onNodeData(builder, s.reqValue[bIndex], data, st)
	s.log.Debugf("onNodeData unresolved(%d), for (%s)\n", unresolved, st)
	s.bMutex[bIndex].Unlock()

	if unresolved == 0 {
		s.Complete(st)
	}
	need := unresolved/configMaxRequestHash + 1
	s.reqUnresolved(st, builder, need)
}

func (s *syncer) onReceive(pi module.ProtocolInfo, b []byte, p *peer) {
	s.processMsg(pi, b, p)
}

func (s *syncer) onJoin(p *peer) {
	s.log.Tracef("onJoin peer(%s)\n", p)
	p.cb = s
	s._requestIfNotEnough(p)
}

func (s *syncer) onLeave(id module.PeerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.log.Tracef("onLeave id(%s)\n", id)
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

func parseMessage(pi module.ProtocolInfo, b []byte) (uint32, interface{}, error) {
	switch pi {
	case protoResult:
		data := new(result)
		if _, err := c.UnmarshalFromBytes(b, data); err != nil {
			return 0, nil, err
		}
		return data.ReqID, data, nil
	case protoNodeData:
		data := new(nodeData)
		if _, err := c.UnmarshalFromBytes(b, data); err != nil {
			return 0, nil, err
		}
		return data.ReqID, data, nil
	default:
		return 0, nil, errors.IllegalArgumentError.Errorf(
			"UnknownProtocol(proto=%d)", pi)
	}
}

func (s *syncer) processMsg(pi module.ProtocolInfo, b []byte, p *peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	rp := s.sentReq[p.id]
	if rp == nil {
		s.log.Tracef("peer(%s) for (%s) is already received\n", p, pi)
		return
	}

	reqID, i, err := parseMessage(pi, b)
	if err != nil || reqID != p.reqID {
		s.log.Infof(
			"Failed onReceive. err(%v), receivedReqID(%d), p.reqID(%d), pi(%s)\n",
			err, reqID, p.reqID, pi)
		return
	}
	p.timer.Stop()
	delete(s.sentReq, p.id)

	go func() {
		if p.onReceive(pi, i) == false {
			s.vpool.push(p)
		}
	}()
}

func (s *syncer) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for k, p := range s.sentReq {
		delete(s.sentReq, k)
		p.timer.Stop()
		s.vpool.push(p)
	}
	s.cond.Broadcast()
	s.finishCh <- errors.ErrInterrupted
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
	s.log.Tracef("vpool(%d), pool(%d)\n", s.vpool.size(), s.pool.size())
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.vpool.size() < configMaxPeerForSync && s.vpool.size() != s.pool.size() {
		err := s.client.hasNode(
			p, s.ah, s.prh,
			s.nrh, s.vlh, s.processMsg)
		if err != nil {
			s.log.Info("Failed to request hasNode to %s, err(%+v)\n", p, err)
			s.ivpool.push(p)
		} else {
			s.sentReq[p.id] = p
		}
	} else {
		s.ivpool.push(p)
	}
}

func (s *syncer) _returnPeers(peers ...*peer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, p := range peers {
		delete(s.sentReq, p.id)
		s.vpool.push(p)
	}

	if s.waitingPeerCnt > 0 {
		s.cond.Signal()
	}
}

func (s *syncer) _reservePeers(need int, st syncType) []*peer {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var peers []*peer
	var size int
	s.waitingPeerCnt += 1
	for {
		if s.complete&st == st {
			s.waitingPeerCnt -= 1
			if s.waitingPeerCnt > 0 {
				s.cond.Signal()
			}
			return nil
		}

		size = s.vpool.size()
		if size == 0 {
			go s._updateValidPool()
			s.log.Tracef("_reservePeers size = %d\n", s.vpool.size())
			s.cond.Wait()
			s.log.Tracef("_reservePeers Wake up peers !!\n")
			continue
		}

		if size > need {
			size = need
		}
		peers = make([]*peer, size)
		for i := 0; i < size; i++ {
			peer := s.vpool.pop()
			s.sentReq[peer.id] = peer
			peers[i] = peer
		}
		s.waitingPeerCnt -= 1
		break
	}

	s.log.Tracef("_reservePeers size = %d, peers = %v\n", size, peers)
	return peers
}

func (s *syncer) onResult(status errCode, p *peer) {
	if status == NoError {
		s._returnPeers(p)
	} else {
		s._requestIfNotEnough(p)
	}
}

func (s *syncer) newMerkleBuilder() merkle.Builder {
	if s.noBuffer {
		return merkle.NewBuilderWithRawDatabase(s.database)
	} else {
		return merkle.NewBuilder(s.database)
	}
}

func (s *syncer) ForceSync() (*Result, error) {
	s.log.Debugln("ForceSync")
	startTime := time.Now()
	s.startTime = startTime
	s.cb(s, true)
	defer func() {
		s.cb(s, false)
		syncDuration := time.Now().Sub(startTime)
		elapsedMS := float64(syncDuration/time.Microsecond) / 1000
		s.log.Infof("ForceSync : Elapsed: %9.3f ms\n", elapsedMS)
	}()

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

	var ess state.ExtensionSnapshot
	if len(s.ed) > 0 {
		eb := s.newMerkleBuilder()
		s.builder[syncExtensionState.toIndex()] = eb
		s.reqValue[syncExtensionState.toIndex()] = make(map[string]bool)
		ess = s.plt.NewExtensionWithBuilder(eb, s.ed)
		go s.reqUnresolved(syncExtensionState, eb, 1)
	} else {
		s.Complete(syncExtensionState)
	}

	builder := s.newMerkleBuilder()
	s.builder[syncWorldState.toIndex()] = builder
	s.reqValue[syncWorldState.toIndex()] = make(map[string]bool)
	if wss, err := state.NewWorldSnapshotWithBuilder(builder, s.ah, s.vlh, ess); err == nil {
		s.wss = wss
	} else {
		return nil, err
	}
	go s.reqUnresolved(syncWorldState, builder, 1)

	rf := func(t syncType, rl *module.ReceiptList, rh []byte) {
		if len(rh) != 0 {
			builder := s.newMerkleBuilder()
			s.builder[t.toIndex()] = builder
			s.reqValue[t.toIndex()] = make(map[string]bool)
			*rl = txresult.NewReceiptListWithBuilder(builder, rh)
			go s.reqUnresolved(t, builder, 1)
		} else {
			*rl = txresult.NewReceiptListFromSlice(s.database, []txresult.Receipt{})
			s.Complete(t)
		}
	}

	rf(syncPatchReceipts, &s.prl, s.prh)
	rf(syncNormalReceipts, &s.nrl, s.nrh)

	if err := <-s.finishCh; err != nil {
		return nil, err
	} else {
		return &Result{s.wss, s.prl, s.nrl}, nil
	}
}

func (s *syncer) Finalize() error {
	s.log.Debugf("Finalize :  ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)\n",
		s.ah, s.prh, s.nrh, s.vlh, s.ed)
	for i, t := range []syncType{syncWorldState, syncPatchReceipts, syncNormalReceipts, syncExtensionState} {
		builder := s.builder[t.toIndex()]
		if builder == nil {
			continue
		} else {
			s.log.Tracef("Flush %s\n", t)
			if err := builder.Flush(true); err != nil {
				s.log.Errorf("Failed to flush for %d builder err(%+v)\n", i, err)
				return err
			}
		}
	}
	syncDuration := time.Now().Sub(s.startTime)
	elapsedMS := float64(syncDuration/time.Microsecond) / 1000
	s.log.Infof("Finalize : Elapsed: %9.3f ms\n", elapsedMS)
	return nil
}

func newSyncer(database db.Database, c *client, p *peerPool, plt Platform,
	accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData []byte,
	log log.Logger, noBuffer bool, cb func(syncer SyncerImpl, syncing bool)) *syncer {
	log.Debugf("newSyncer ah(%#x), prh(%#x), nrh(%#x), vlh(%#x), ed(%#x)",
		accountsHash, pReceiptsHash, nReceiptsHash, validatorListHash, extensionData)

	s := &syncer{
		database: database,
		pool:     p,
		client:   c,
		plt:      plt,
		noBuffer: noBuffer,
		vpool:    newPeerPool(),
		ivpool:   newPeerPool(),
		sentReq:  make(map[module.PeerID]*peer),
		ah:       accountsHash,
		prh:      pReceiptsHash,
		nrh:      nReceiptsHash,
		vlh:      validatorListHash,
		ed:       extensionData,
		finishCh: make(chan error, 1),
		log:      log,
		cb:       cb,
	}
	s.cond = sync.NewCond(&s.mutex)
	return s
}
