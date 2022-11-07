package consensus

import (
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/consensus/fastsync"
	"github.com/icon-project/goloop/module"
)

const (
	configSendBPS                   = -1
	configRoundStateMessageInterval = 300 * time.Millisecond
	configFastSyncThreshold         = 4
)

type Engine interface {
	fastsync.BlockProofProvider

	GetCommitBlockParts(h int64) PartSet
	GetCommitPrecommits(h int64) *voteList
	GetPrecommits(r int32) *voteList
	// GetVotes returns union of a set of prevotes pv(i) where
	// pvMask.Get(i) == 0 and a set of precommits pc(i) where
	// pcMask.Get(i) == 0. For example, if the all bits for mask is 1,
	// no votes are returned.
	GetVotes(r int32, pvMask *bitArray, pcMask *bitArray) *voteList
	GetRoundState() *peerRoundState

	Height() int64
	Round() int32
	Step() step

	ReceiveBlockPartMessage(msg *BlockPartMessage, unicast bool) (int, error)
	ReceiveVoteListMessage(msg *voteListMessage, unicast bool) error
	ReceiveBlock(br fastsync.BlockResult)
}

type Syncer interface {
	Start() error
	Stop()
	OnEngineStepChange()
}

var SyncerProtocols = []module.ProtocolInfo{
	ProtoBlockPart,
	ProtoRoundState,
	ProtoVoteList,
}

type peer struct {
	*syncer
	id         module.PeerID
	wakeUpChan chan struct{}
	stopped    chan struct{}
	log        log.Logger

	running bool
	*peerRoundState
}

func newPeer(syncer *syncer, id module.PeerID) *peer {
	peerLogger := syncer.log.WithFields(log.Fields{
		"peer": common.HexPre(id.Bytes()),
	})
	return &peer{
		syncer:     syncer,
		id:         id,
		wakeUpChan: make(chan struct{}, 1),
		stopped:    make(chan struct{}),
		log:        peerLogger,
		running:    true, // TODO better way
	}
}

func (p *peer) setRoundState(prs *peerRoundState) {
	p.peerRoundState = prs
	p.wakeUp()
}

func (p *peer) doSync() (module.ProtocolInfo, Message) {
	e := p.engine
	if p.peerRoundState == nil {
		p.log.Tracef("nil peer round state\n")
		return 0, nil
	}

	if !p.peerRoundState.Sync {
		p.log.Tracef("peer round state: no sync\n")
		return 0, nil
	}

	if p.Height < e.Height() || (p.Height == e.Height() && e.Step() >= stepCommit) {
		if p.BlockPartsMask == nil {
			var vl *voteList
			if p.Height == e.Height() {
				// send prevotes to prevent the peer from entering precommit
				// without polka and sending nil precommit
				vl = e.GetVotes(e.Round(), p.PrevotesMask, p.PrecommitsMask)
			} else {
				vl = e.GetCommitPrecommits(p.Height)
			}
			if vl == nil {
				return 0, nil
			}
			msg := newVoteListMessage()
			msg.VoteList = vl
			partSet := e.GetCommitBlockParts(p.Height)
			if partSet == nil {
				return 0, nil
			}
			p.BlockPartsMask = newBitArray(partSet.Parts())
			p.log.Tracef("PC for commit %v\n", p.Height)
			return ProtoVoteList, msg
		}
		partSet := e.GetCommitBlockParts(p.Height)
		if partSet == nil {
			return 0, nil
		}
		mask := p.BlockPartsMask.Copy()
		mask.Flip()
		mask.AssignAnd(partSet.GetMask())
		idx := mask.PickRandom()
		if idx < 0 {
			p.log.Tracef("no bp to send: %v/%v\n", p.BlockPartsMask, partSet.GetMask())
			return 0, nil
		}
		part := partSet.GetPart(idx)
		msg := newBlockPartMessage()
		msg.Height = p.Height
		msg.Index = uint16(idx)
		msg.BlockPart = part.Bytes()
		p.BlockPartsMask.Set(idx)
		return ProtoBlockPart, msg
	}
	if p.Height > e.Height() {
		p.log.Tracef("higher peer height %v > %v\n", p.Height, e.Height())
		if p.Height > e.Height()+configFastSyncThreshold && p.syncer.fetchCanceler == nil {
			p.syncer.fetchCanceler, _ = p.syncer.fsm.FetchBlocks(e.Height(), -1, p.syncer)
		}
		return 0, nil
	}

	if p.Round < e.Round() && e.Step() >= stepPrevoteWait {
		vl := e.GetVotes(e.Round(), p.PrevotesMask, p.PrecommitsMask)
		msg := newVoteListMessage()
		msg.VoteList = vl
		p.peerRoundState = nil
		p.log.Tracef("Votes for round %v\n", e.Round())
		return ProtoVoteList, msg
	} else if p.Round < e.Round() {
		vl := e.GetVotes(e.Round()-1, p.PrevotesMask, p.PrecommitsMask)
		msg := newVoteListMessage()
		msg.VoteList = vl
		p.peerRoundState = nil
		p.log.Tracef("Votes for prev round %v\n", e.Round()-1)
		return ProtoVoteList, msg
	} else if p.Round == e.Round() {
		rs := e.GetRoundState()
		p.log.Tracef("r=%v pv=%v/%v pc=%v/%v\n", e.Round(), p.PrevotesMask, rs.PrevotesMask, p.PrecommitsMask, rs.PrecommitsMask)
		vl := e.GetVotes(e.Round(), p.PrevotesMask, p.PrecommitsMask)
		if vl.Len() > 0 {
			msg := newVoteListMessage()
			msg.VoteList = vl
			p.peerRoundState = nil
			return ProtoVoteList, msg
		}
	}

	p.log.Tracef("nothing to send\n")
	return 0, nil
}

func (p *peer) sync() {
	var nextSendTime *time.Time

	p.log.Debugf("peer start sync\n")
	for {
		<-p.wakeUpChan

		p.log.Tracef("peer.wakeUp\n")
		p.mutex.Lock()
		if !p.running {
			p.mutex.Unlock()
			p.log.Tracef("peer is not running\n")
			p.stopped <- struct{}{}
			break
		}
		now := time.Now()
		if nextSendTime != nil && now.Before(*nextSendTime) {
			p.mutex.Unlock()
			p.log.Tracef("peer.now=%v nextSendTime=%v\n", now.Format(time.StampMicro), nextSendTime.Format(time.StampMicro))
			continue
		}
		proto, msg := p.doSync()
		p.mutex.Unlock()

		if msg == nil {
			nextSendTime = nil
			continue
		}

		msgBS, err := msgCodec.MarshalToBytes(msg)
		if err != nil {
			p.log.Panicf("peer.sync: %v\n", err)
		}
		p.log.Debugf("sendMessage %v\n", msg)
		if err = p.ph.Unicast(proto, msgBS, p.id); err != nil {
			p.log.Warnf("peer.sync: %v\n", err)
		}
		if configSendBPS < 0 {
			p.wakeUp()
			continue
		}
		if nextSendTime == nil {
			nextSendTime = &now
		}
		delta := time.Second * time.Duration(len(msgBS)) / configSendBPS
		next := nextSendTime.Add(delta)
		nextSendTime = &next
		waitTime := nextSendTime.Sub(now)
		p.log.Tracef("msg size=%v delta=%v waitTime=%v\n", len(msgBS), delta, waitTime)
		if waitTime > time.Duration(0) {
			time.AfterFunc(waitTime, func() {
				p.wakeUp()
			})
		} else {
			p.wakeUp()
		}
	}
}

func (p *peer) stop() {
	p.running = false
	p.wakeUp()
	p.syncer.mutex.CallAfterUnlock(func() {
		<-p.stopped
	})
}

func (p *peer) wakeUp() {
	select {
	case p.wakeUpChan <- struct{}{}:
	default:
	}
}

type syncer struct {
	engine Engine
	log    log.Logger
	nm     module.NetworkManager
	bm     module.BlockManager
	mutex  *common.Mutex
	addr   module.Address
	fsm    fastsync.Manager

	ph            module.ProtocolHandler
	peers         []*peer
	timer         *time.Timer
	lastSendTime  time.Time
	running       bool
	fetchCanceler func() bool
}

func newSyncer(e Engine, logger log.Logger, nm module.NetworkManager, bm module.BlockManager, mutex *common.Mutex, addr module.Address) (Syncer, error) {
	fsm, err := fastsync.NewManager(nm, bm, e, logger)
	if err != nil {
		return nil, err
	}
	fsm.StartServer()
	return &syncer{
		engine: e,
		log:    logger,
		nm:     nm,
		bm:     bm,
		mutex:  mutex,
		addr:   addr,
		fsm:    fsm,
	}, nil
}

func (s *syncer) Start() error {
	var err error
	s.ph, err = s.nm.RegisterReactor("consensus.sync", module.ProtoConsensusSync, s, SyncerProtocols, ConfigSyncerPriority, module.NotRegisteredProtocolPolicyClose)
	if err != nil {
		return err
	}

	peerIDs := s.ph.GetPeers()
	s.peers = make([]*peer, len(peerIDs))
	for i, peerID := range peerIDs {
		s.log.Debugf("Start: starting peer list %v\n", common.HexPre(peerID.Bytes()))
		s.peers[i] = newPeer(s, peerID)
		go s.peers[i].sync()
	}

	s.sendRoundStateMessage()
	s.running = true
	return nil
}

func (s *syncer) OnReceive(sp module.ProtocolInfo, bs []byte,
	id module.PeerID) (bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return false, nil
	}

	msg, err := UnmarshalMessage(sp.Uint16(), bs)
	if err != nil {
		s.log.Warnf("OnReceive: error=%+v\n", err)
		return false, err
	}
	s.log.Debugf("OnReceive %v From:%v\n", msg, common.HexPre(id.Bytes()))
	if err := msg.Verify(); err != nil {
		return false, err
	}
	var idx int
	switch m := msg.(type) {
	case *BlockPartMessage:
		idx, err = s.engine.ReceiveBlockPartMessage(m, true)
		if idx < 0 && err != nil {
			return false, err
		}
		for _, p := range s.peers {
			// TODO check mask for optimization
			if p.peerRoundState != nil && p.Height == m.Height &&
				p.Height == p.engine.Height() && p.BlockPartsMask != nil {
				p.wakeUp()
			}
		}
	case *RoundStateMessage:
		for _, p := range s.peers {
			if p.id.Equal(id) {
				p.setRoundState(&m.peerRoundState)
			}
		}
	case *voteListMessage:
		err = s.engine.ReceiveVoteListMessage(m, true)
		if err != nil {
			return false, err
		}
		rs := s.engine.GetRoundState()
		s.log.Tracef("roundState=%+v\n", *rs)
	default:
		s.log.Warnf("received unknown message %v\n", msg)
	}
	return true, nil
}

func (s *syncer) OnFailure(
	err error,
	pi module.ProtocolInfo,
	b []byte,
) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.log.Debugf("OnFailure: subprotocol:%v err:%+v\n", pi, err)

	if !s.running {
		return
	}
}

func (s *syncer) OnJoin(id module.PeerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.log.Debugf("OnJoin: %v\n", common.HexPre(id.Bytes()))

	if !s.running {
		return
	}

	for _, p := range s.peers {
		if p.id.Equal(id) {
			return
		}
	}
	p := newPeer(s, id)
	s.peers = append(s.peers, p)
	go p.sync()
	s.doSendRoundStateMessage(id)
}

func (s *syncer) OnLeave(id module.PeerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.log.Debugf("OnLeave: %v\n", common.HexPre(id.Bytes()))

	if !s.running {
		return
	}

	for i, p := range s.peers {
		if p.id.Equal(id) {
			last := len(s.peers) - 1
			s.peers[i] = s.peers[last]
			s.peers[last] = nil
			s.peers = s.peers[:last]
			p.stop()
			return
		}
	}
}

func (s *syncer) OnEngineStepChange() {
	if !s.running {
		return
	}
	e := s.engine
	if e.Step() == stepPrecommitWait || e.Step() == stepCommit {
		for _, p := range s.peers {
			if p.peerRoundState != nil {
				p.wakeUp()
			}
		}
	}

	send := e.Step() == stepTransactionWait ||
		(e.Round() > 0 && e.Step() == stepPropose) ||
		e.Step() == stepCommit
	if send {
		s.sendRoundStateMessage()
	}
}

func (s *syncer) doSendRoundStateMessage(id module.PeerID) {
	e := s.engine
	msg := newRoundStateMessage()
	msg.peerRoundState = *e.GetRoundState()
	bs, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		s.log.Panicf("doSendRoundStateMessage: %+v\n", err)
	}
	if id == nil {
		if len(s.peers) > 0 {
			s.log.Debugf("neighborcastRoundState %v\n", msg)
			err = s.ph.Broadcast(ProtoRoundState, bs, module.BROADCAST_NEIGHBOR)
		}
	} else {
		s.log.Debugf("sendRoundState %v To:%v\n", msg, common.HexPre(id.Bytes()))
		err = s.ph.Unicast(ProtoRoundState, bs, id)
	}
	if err != nil {
		s.log.Warnf("doSendRoundStateMessage: %+v\n", err)
	}
}

func (s *syncer) sendRoundStateMessage() {
	s.doSendRoundStateMessage(nil)
	s.lastSendTime = time.Now()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	if s.engine.Step() == stepNewHeight {
		return
	}

	var timer *time.Timer
	timer = time.AfterFunc(configRoundStateMessageInterval, func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if s.timer != timer {
			return
		}

		s.sendRoundStateMessage()
	})
	s.timer = timer
}

func (s *syncer) Stop() {
	for _, p := range s.peers {
		p.stop()
	}

	s.running = false

	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.fsm.StopServer()
	if s.fetchCanceler != nil {
		s.fetchCanceler()
		s.fetchCanceler = nil
	}
	s.fsm.Term()
}

func (s *syncer) OnBlock(br fastsync.BlockResult) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	s.log.Debugf("syncer.OnBlock %d\n", br.Block().Height())
	s.engine.ReceiveBlock(br)
}

func (s *syncer) OnEnd(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}

	s.log.Debugf("syncer.OnEnd %+v\n", err)
	s.fetchCanceler = nil
}
