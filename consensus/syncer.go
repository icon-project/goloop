package consensus

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	configSendBPS                   = 1024 * 300
	configRoundStateMessageInterval = 250 * time.Millisecond
)

type Engine interface {
	GetCommitBlockParts(h int64) PartSet
	GetCommitPrecommits(h int64) *roundVoteList
	GetPrecommits(r int32) *roundVoteList
	GetVotes(r int32, prevotesMask *bitArray, precommitsMask *bitArray) *roundVoteList
	GetRoundState() *peerRoundState

	Height() int64
	Round() int32
	Step() step

	ReceiveBlockPartMessage(msg *blockPartMessage, unicast bool) (int, error)
	ReceiveVoteMessage(msg *voteMessage, unicast bool) (int, error)
}

type Syncer interface {
	Start() error
	Stop()
	OnEngineStepChange()
}

var syncerProtocols = []module.ProtocolInfo{
	protoBlockPart,
	protoRoundState,
	protoVoteList,
}

type peer struct {
	*syncer
	id         module.PeerID
	wakeUpChan chan struct{}

	running bool
	*peerRoundState
}

func newPeer(syncer *syncer, id module.PeerID) *peer {
	return &peer{
		syncer:     syncer,
		id:         id,
		wakeUpChan: make(chan struct{}, 1),
		running:    true, // TODO better way
	}
}

func (p *peer) setRoundState(prs *peerRoundState) {
	p.peerRoundState = prs
	p.wakeUp()
}

func (p *peer) doSync() (module.ProtocolInfo, message) {
	e := p.engine
	if p.peerRoundState == nil {
		logger.Printf("nil peer round state\n")
		return nil, nil
	}

	if p.Height < e.Height() || (p.Height == e.Height() && e.Step() >= stepCommit) {
		if p.BlockPartsMask == nil {
			vl := e.GetCommitPrecommits(p.Height)
			msg := newVoteListMessage()
			msg.VoteList = vl
			p.BlockPartsMask = newBitArray(e.GetCommitBlockParts(p.Height).Parts())
			p.BlockPartsMask.Flip()
			logger.Printf("vote list for commit\n")
			return protoVoteList, msg
		}
		partSet := e.GetCommitBlockParts(p.Height)
		var mask *bitArray
		if partSet.IsComplete() {
			mask = p.BlockPartsMask
		} else {
			mask = p.BlockPartsMask.Copy()
			mask.AssignAnd(partSet.GetMask())
		}
		idx := mask.PickRandom()
		if idx < 0 {
			logger.Printf("no bp to send\n")
			return nil, nil
		}
		part := partSet.GetPart(idx)
		msg := newBlockPartMessage()
		msg.Height = p.Height
		msg.BlockPart = part.Bytes()
		p.BlockPartsMask.Unset(idx)
		logger.Printf("bp to send\n")
		return protoBlockPart, msg
	}
	if p.Height > e.Height() {
		return nil, nil
	}

	if p.Round < e.Round() && e.Step() >= stepPrecommitWait {
		vl := e.GetPrecommits(e.Round())
		msg := newVoteListMessage()
		msg.VoteList = vl
		p.peerRoundState = nil
		logger.Printf("vl for PC\n")
		return protoVoteList, msg
	} else if p.Round < e.Round()-1 {
		vl := e.GetPrecommits(e.Round() - 1)
		msg := newVoteListMessage()
		msg.VoteList = vl
		p.peerRoundState = nil
		logger.Printf("vl for PC(R-1)\n")
		return protoVoteList, msg
	} else if p.Round == e.Round() {
		logger.Printf("r=%v pv=%v pc=%v\n", e.Round(), p.PrevotesMask, p.PrecommitsMask)
		vl := e.GetVotes(e.Round(), p.PrevotesMask, p.PrecommitsMask)
		if vl.Len() > 0 {
			msg := newVoteListMessage()
			msg.VoteList = vl
			p.peerRoundState = nil
			logger.Printf("vl for current round votes\n")
			return protoVoteList, msg
		}
		logger.Printf("empty vl to send\n")
	}

	logger.Printf("nothing to send\n")
	return nil, nil
}

func (p *peer) sync() {
	var nextSendTime *time.Time

	logger.Printf("%x| peer start sync\n", p.id.Bytes()[1:3])
	for {
		<-p.wakeUpChan

		logger.Printf("%x| peer.wakeUp\n", p.id.Bytes()[1:3])
		p.mutex.Lock()
		if !p.running {
			p.mutex.Unlock()
			logger.Printf("%x| peer.!running\n", p.id.Bytes()[1:3])
			break
		}
		now := time.Now()
		if nextSendTime != nil && now.Before(*nextSendTime) {
			p.mutex.Unlock()
			logger.Printf("%x| peer.now=%v nextSendTime=%v\n", p.id.Bytes()[1:3], now, nextSendTime)
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
			logger.Panicf("peer.sync: %v\n", err)
		}
		logger.Printf("send message %+v\n", msg)
		if err = p.ph.Unicast(proto, msgBS, p.id); err != nil {
			logger.Printf("peer.sync: %v\n", err)
		}
		if nextSendTime == nil {
			nextSendTime = &now
		}
		delta := time.Second * time.Duration(len(msgBS)) / configSendBPS
		next := nextSendTime.Add(delta)
		nextSendTime = &next
		waitTime := now.Sub(*nextSendTime)
		time.AfterFunc(waitTime, func() {
			p.wakeUp()
		})
	}
}

func (p *peer) stop() {
	p.running = false
	p.wakeUp()
}

func (p *peer) wakeUp() {
	select {
	case p.wakeUpChan <- struct{}{}:
	default:
	}
}

type syncer struct {
	engine Engine
	nm     module.NetworkManager
	mutex  *sync.Mutex

	ph           module.ProtocolHandler
	peers        []*peer
	timer        *time.Timer
	lastSendTime time.Time
	running      bool
}

func newSyncer(e Engine, nm module.NetworkManager, mutex *sync.Mutex) Syncer {
	return &syncer{
		engine: e,
		nm:     nm,
		mutex:  mutex,
	}
}

func (s *syncer) Start() error {
	var err error
	s.ph, err = s.nm.RegisterReactor("consensus.sync", s, syncerProtocols, configSyncerPriority)
	if err != nil {
		return err
	}

	peerIDs := s.nm.GetPeers()
	s.peers = make([]*peer, len(peerIDs))
	for i, peerID := range peerIDs {
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

	msg, err := unmarshalMessage(sp, bs)
	if err != nil {
		logger.Printf("OnReceive: error=%v\n", err)
		return false, err
	}
	logger.Printf("OnReceive %+v\n", msg)
	if err := msg.verify(); err != nil {
		return false, err
	}
	var idx int
	switch m := msg.(type) {
	case *blockPartMessage:
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
	case *roundStateMessage:
		for _, p := range s.peers {
			if p.id.Equal(id) {
				m.peerRoundState.PrevotesMask.Flip()
				m.peerRoundState.PrecommitsMask.Flip()
				if m.peerRoundState.BlockPartsMask != nil {
					m.peerRoundState.BlockPartsMask.Flip()
				}
				p.setRoundState(&m.peerRoundState)
			}
		}
	case *voteListMessage:
		for i := 0; i < m.VoteList.Len(); i++ {
			s.engine.ReceiveVoteMessage(m.VoteList.Get(i), true)
		}
	default:
		logger.Panicf("received unknown message %v\n", msg)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *syncer) OnError(
	err error,
	pi module.ProtocolInfo,
	b []byte,
	id module.PeerID,
) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return
	}
}

func (s *syncer) OnJoin(id module.PeerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

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
	if e.Step() == stepPropose || e.Step() == stepPrecommitWait || e.Step() == stepCommit {
		s.sendRoundStateMessage()
	}
}

func (s *syncer) doSendRoundStateMessage(id module.PeerID) {
	e := s.engine
	msg := newRoundStateMessage()
	msg.peerRoundState = *e.GetRoundState()
	bs, err := msgCodec.MarshalToBytes(msg)
	if err != nil {
		logger.Panicf("syncer.doSendRoundStateMessage: %v\n", err)
	}
	if id == nil {
		logger.Printf("broadcastRoundState : %+v\n", msg)
		s.ph.Broadcast(protoRoundState, bs, module.BROADCAST_NEIGHBOR)
	} else {
		logger.Printf("sendRoundState : %+v\n", msg)
		s.ph.Unicast(protoRoundState, bs, id)
	}
}

func (s *syncer) sendRoundStateMessage() {
	s.doSendRoundStateMessage(nil)
	s.lastSendTime = time.Now()
	if s.timer != nil {
		s.timer.Stop()
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

	s.timer.Stop()
	s.timer = nil
	s.running = false
}
