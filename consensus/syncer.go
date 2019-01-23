package consensus

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	configSendBPS                   = 1024 * 500
	configRoundStateMessageInterval = 300 * time.Millisecond
)

type Engine interface {
	GetCommitBlockParts(h int64) PartSet
	GetCommitPrecommits(h int64) *voteList
	GetPrecommits(r int32) *voteList
	GetVotes(r int32, prevotesMask *bitArray, precommitsMask *bitArray) *voteList
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
	logger     *log.Logger
	debug      *log.Logger

	running bool
	*peerRoundState
}

func newPeer(syncer *syncer, id module.PeerID) *peer {
	prefix := fmt.Sprintf("%x|CS|%x|", syncer.addr.Bytes()[1:3], id.Bytes()[1:3])
	return &peer{
		syncer:     syncer,
		id:         id,
		wakeUpChan: make(chan struct{}, 1),
		logger:     log.New(os.Stderr, prefix, log.Lshortfile|log.Lmicroseconds),
		debug:      log.New(debugWriter, prefix, log.Lshortfile|log.Lmicroseconds),
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
		p.debug.Printf("nil peer round state\n")
		return nil, nil
	}

	if p.Height < e.Height() || (p.Height == e.Height() && e.Step() >= stepCommit) {
		if p.BlockPartsMask == nil {
			vl := e.GetCommitPrecommits(p.Height)
			msg := newVoteListMessage()
			msg.VoteList = vl
			p.BlockPartsMask = newBitArray(e.GetCommitBlockParts(p.Height).Parts())
			p.debug.Printf("PC for commit %v\n", p.Height)
			return protoVoteList, msg
		}
		partSet := e.GetCommitBlockParts(p.Height)
		mask := p.BlockPartsMask.Copy()
		mask.Flip()
		mask.AssignAnd(partSet.GetMask())
		idx := mask.PickRandom()
		if idx < 0 {
			p.debug.Printf("no bp to send: %v/%v\n", p.BlockPartsMask, partSet.GetMask())
			return nil, nil
		}
		part := partSet.GetPart(idx)
		msg := newBlockPartMessage()
		msg.Height = p.Height
		msg.Index = uint16(idx)
		msg.BlockPart = part.Bytes()
		p.BlockPartsMask.Set(idx)
		return protoBlockPart, msg
	}
	if p.Height > e.Height() {
		p.debug.Printf("higher peer height %v > %v\n", p.Height, e.Height())
		return nil, nil
	}

	if p.Round < e.Round() && e.Step() >= stepPrecommitWait {
		vl := e.GetPrecommits(e.Round())
		msg := newVoteListMessage()
		msg.VoteList = vl
		p.peerRoundState = nil
		p.debug.Printf("PC for round %v\n", e.Round())
		return protoVoteList, msg
	} else if p.Round < e.Round() {
		// TODO: check peer step
		vl := e.GetPrecommits(e.Round() - 1)
		msg := newVoteListMessage()
		msg.VoteList = vl
		p.peerRoundState = nil
		p.debug.Printf("PC for round %v (prev round)\n", e.Round())
		return protoVoteList, msg
	} else if p.Round == e.Round() {
		rs := e.GetRoundState()
		p.debug.Printf("r=%v pv=%v/%v pc=%v/%v\n", e.Round(), p.PrevotesMask, rs.PrevotesMask, p.PrecommitsMask, rs.PrecommitsMask)
		pv := p.PrevotesMask.Copy()
		pv.Flip()
		pc := p.PrecommitsMask.Copy()
		pc.Flip()
		vl := e.GetVotes(e.Round(), pv, pc)
		if vl.Len() > 0 {
			msg := newVoteListMessage()
			msg.VoteList = vl
			p.peerRoundState = nil
			return protoVoteList, msg
		}
	}

	p.debug.Printf("nothing to send\n")
	return nil, nil
}

func (p *peer) sync() {
	var nextSendTime *time.Time

	p.logger.Printf("peer start sync\n")
	for {
		<-p.wakeUpChan

		p.debug.Printf("peer.wakeUp\n")
		p.mutex.Lock()
		if !p.running {
			p.mutex.Unlock()
			p.debug.Printf("peer is not running\n")
			break
		}
		now := time.Now()
		if nextSendTime != nil && now.Before(*nextSendTime) {
			p.mutex.Unlock()
			p.debug.Printf("peer.now=%v nextSendTime=%v\n", now.Format(time.StampMicro), nextSendTime.Format(time.StampMicro))
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
			p.logger.Panicf("peer.sync: %v\n", err)
		}
		p.logger.Printf("send message %+v\n", msg)
		if err = p.ph.Unicast(proto, msgBS, p.id); err != nil {
			p.logger.Printf("peer.sync: %v\n", err)
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
		p.debug.Printf("msg size=%v delta=%v waitTime=%v\n", len(msgBS), delta, waitTime)
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
	addr   module.Address

	ph           module.ProtocolHandler
	peers        []*peer
	timer        *time.Timer
	lastSendTime time.Time
	running      bool
}

func newSyncer(e Engine, nm module.NetworkManager, mutex *sync.Mutex, addr module.Address) Syncer {
	return &syncer{
		engine: e,
		nm:     nm,
		mutex:  mutex,
		addr:   addr,
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
		logger.Printf("Start: starting peer list %v\n", peerID)
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

	msg, err := unmarshalMessage(sp.Uint16(), bs)
	if err != nil {
		logger.Printf("OnReceive: error=%+v\n", err)
		return false, err
	}
	logger.Printf("OnReceive %+v from %x\n", msg, id.Bytes()[1:3])
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
				p.setRoundState(&m.peerRoundState)
			}
		}
	case *voteListMessage:
		for i := 0; i < m.VoteList.Len(); i++ {
			s.engine.ReceiveVoteMessage(m.VoteList.Get(i), true)
		}
		rs := s.engine.GetRoundState()
		debug.Printf("roundState=%+v\n", *rs)
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

	logger.Printf("OnJoin: %v\n", id)

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

	logger.Printf("OnLeave: %v\n", id)

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
		logger.Panicf("syncer.doSendRoundStateMessage: %+v\n", err)
	}
	if id == nil {
		if len(s.peers) > 0 {
			logger.Printf("broadcastRoundState : %+v\n", msg)
			err = s.ph.Broadcast(protoRoundState, bs, module.BROADCAST_NEIGHBOR)
		}
	} else {
		logger.Printf("sendRoundState : %+v\n", msg)
		err = s.ph.Unicast(protoRoundState, bs, id)
	}
	if err != nil {
		logger.Printf("syncer.doSendRoundStateMessage: %+v\n", err)
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
