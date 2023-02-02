package network

import (
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	configPeerAckTimeout = 1000 * time.Millisecond
	configAckWait        = 500 * time.Microsecond
)

type streamMessage struct {
	Seq     uint16
	Ack     uint16
	Payload []byte
	// TODO: NAcks to reduce retransmitted traffics
}

type streamReactor struct {
	sync.Mutex
	clock       common.Clock
	userReactor module.Reactor
	streamPI    module.ProtocolInfo // streamPI is used if payload is nil

	ph      module.ProtocolHandler
	streams []*stream
}

type sendItem struct {
	pi module.ProtocolInfo
	b  []byte
	t  time.Time
}

type stream struct {
	r  *streamReactor
	id module.PeerID

	seq          uint16
	peerAck      uint16
	timedout     bool
	repostTimer  *common.Timer
	unackedItems []*sendItem // in seq order. first item's seq is peerAck+1
	peerSeq      uint16
	ackPostTimer *common.Timer // nil if nothing to ack
}

func newReactor(clock common.Clock, ur module.Reactor,
	spi module.ProtocolInfo) *streamReactor {
	return &streamReactor{
		clock:       clock,
		userReactor: ur,
		streamPI:    spi,
	}
}

func (r *streamReactor) dispose() {
	for _, s := range r.streams {
		s.dispose()
	}
}

func (r *streamReactor) streamForPeer(id module.PeerID) *stream {
	for _, s := range r.streams {
		if s.id.Equal(id) {
			return s
		}
	}
	return nil
}

func (r *streamReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	var payload []byte
	var err error
	consume := func() bool {
		r.Lock()
		defer r.Unlock()

		s := r.streamForPeer(id)
		if s == nil {
			return true
		}
		sm := &streamMessage{}
		_, e := codec.UnmarshalFromBytes(b, sm)
		if e != nil {
			err = e
			return true
		}
		if !s.receive(sm) {
			return true
		}
		if sm.Payload == nil {
			return true
		}
		payload = sm.Payload
		return false
	}()
	if consume {
		return false, err
	}

	return r.userReactor.OnReceive(pi, payload, id)
}

func (r *streamReactor) OnJoin(id module.PeerID) {
	r.Lock()
	defer r.userReactor.OnJoin(id)
	defer r.Unlock()

	if r.streamForPeer(id) == nil {
		r.streams = append(r.streams, newStream(r, id))
	}
}

func (r *streamReactor) OnLeave(id module.PeerID) {
	r.Lock()
	defer r.userReactor.OnLeave(id)
	defer r.Unlock()

	for i, s := range r.streams {
		if s.id.Equal(id) {
			last := len(r.streams) - 1
			r.streams[i] = r.streams[last]
			r.streams[last] = nil
			r.streams = r.streams[:last]
			s.dispose()
			break
		}
	}
}

func (r *streamReactor) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	return errors.Errorf("Broadcast is not supported for stream")
}

func (r *streamReactor) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	return errors.Errorf("Multicast is not supported for stream")
}

func (r *streamReactor) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	r.Lock()
	defer r.Unlock()

	s := r.streamForPeer(id)
	if s == nil {
		return errors.Errorf("Unknown peer %s", id)
	}
	return s.send(pi, b)
}

func (r *streamReactor) GetPeers() []module.PeerID {
	return r.ph.GetPeers()
}

func newStream(r *streamReactor, id module.PeerID) *stream {
	return &stream{
		r:  r,
		id: id,
	}
}

func (s *stream) dispose() {
	if s.repostTimer != nil {
		s.repostTimer.Stop()
		s.repostTimer = nil
	}
	if s.ackPostTimer != nil {
		s.ackPostTimer.Stop()
		s.ackPostTimer = nil
	}
}

func (s *stream) postMessage(pi module.ProtocolInfo, sm *streamMessage) error {
	bs := codec.MustMarshalToBytes(&sm)
	err := s.r.ph.Unicast(pi, bs, s.id)
	if err == nil {
		if s.ackPostTimer != nil {
			s.ackPostTimer.Stop()
			s.ackPostTimer = nil
		}
	}
	return err
}

func (s *stream) postAck() error {
	return s.postMessage(s.r.streamPI, &streamMessage{
		Seq: s.seq,
		Ack: s.peerSeq,
	})
}

func (s *stream) receive(sm *streamMessage) (res bool) {
	if sm.Seq == s.peerSeq+1 {
		s.peerSeq++
		if s.ackPostTimer == nil {
			var timer common.Timer
			timer = s.r.clock.AfterFunc(configAckWait, func() {
				s.r.Lock()
				defer s.r.Unlock()
				if s.ackPostTimer != &timer {
					return
				}

				s.postAck()
			})
			s.ackPostTimer = &timer
		}
		res = true
	} else {
		// maybe incoming message or outgoing ack was dropped
		if sm.Payload != nil {
			s.postAck()
		}
	}

	// unwrap as seq and ack can wrap
	peerAck := uint32(s.peerAck)
	seq := peerAck + uint32(len(s.unackedItems))
	ack := uint32(sm.Ack)
	if ack < peerAck {
		ack |= 0x10000
	}
	if peerAck < ack && ack <= seq {
		mis := make([]*sendItem, seq-ack)
		copy(mis, s.unackedItems[ack-peerAck:])
		s.unackedItems = mis
		s.peerAck = sm.Ack
		now := s.r.clock.Now()
		for i, si := range s.unackedItems {
			// TODO: retry if it is temporary error
			if now.Equal(si.t) || now.After(si.t) || s.timedout {
				s.postMessage(si.pi, &streamMessage{
					Seq:     s.peerAck + uint16(i) + 1,
					Ack:     s.peerSeq,
					Payload: si.b,
				})
				si.t = now.Add(configPeerAckTimeout)
			}
		}
		s.timedout = false
		if s.repostTimer != nil {
			s.repostTimer.Stop()
		}
		s.updateRepostTimer(now)
	}
	return
}

func (s *stream) send(pi module.ProtocolInfo, b []byte) error {
	nextSeq := s.seq + 1
	if nextSeq == s.peerAck {
		return errors.Errorf("seq wrap")
	}
	if !s.timedout {
		err := s.postMessage(pi, &streamMessage{
			Seq:     nextSeq,
			Ack:     s.peerSeq,
			Payload: b,
		})
		if err != nil {
			return err
		}
	}
	now := s.r.clock.Now()
	p := &sendItem{
		pi: pi,
		b:  b,
		t:  now.Add(configPeerAckTimeout),
	}
	s.seq = nextSeq
	s.unackedItems = append(s.unackedItems, p)
	if s.repostTimer == nil {
		s.updateRepostTimer(now)
	}
	return nil
}

func (s *stream) updateRepostTimer(now time.Time) {
	if len(s.unackedItems) == 0 {
		s.repostTimer = nil
		return
	}
	seq := s.peerAck + 1
	var timer common.Timer
	mi := s.unackedItems[0]
	diff := mi.t.Sub(now)
	timer = s.r.clock.AfterFunc(diff, func() {
		s.r.Lock()
		defer s.r.Unlock()
		if s.repostTimer != &timer {
			return
		}

		s.timedout = true
		now := s.r.clock.Now()
		// TODO: retry if it is temporary error
		s.postMessage(mi.pi, &streamMessage{
			Seq:     seq,
			Ack:     s.peerSeq,
			Payload: mi.b,
		})
		mi.t = now.Add(configPeerAckTimeout)
		s.updateRepostTimer(now)
	})
	s.repostTimer = &timer
}

// for test
func (s *stream) setSeqByForce(seq uint16) {
	s.seq = seq
	s.peerAck = seq
}

func (s *stream) setPeerSeqByForce(seq uint16) {
	s.peerSeq = seq
}

func registerReactorForStreams(nm module.NetworkManager, name string, pi module.ProtocolInfo, ureactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy, clock common.Clock) (*streamReactor, error) {
	r := newReactor(clock, ureactor, piList[0])
	r.Lock()
	defer r.Unlock()

	ph, err := nm.RegisterReactor(name, pi, r, piList, priority, policy)
	if err != nil {
		return nil, err
	}
	r.ph = ph
	for _, id := range ph.GetPeers() {
		r.streams = append(r.streams, newStream(r, id))
	}
	return r, err
}

func (m *manager) tryUnregisterStreamReactor(reactor module.Reactor) *streamReactor {
	for i, sr := range m.streamReactors {
		if sr.userReactor == reactor {
			last := len(m.streamReactors) - 1
			m.streamReactors[i] = m.streamReactors[last]
			m.streamReactors[last] = nil
			m.streamReactors = m.streamReactors[:last]
			sr.dispose()
			return sr
		}
	}
	return nil
}

func (m *manager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	r, err := registerReactorForStreams(m, name, pi, reactor, piList, priority, policy, &common.GoTimeClock{})
	if err != nil {
		return r, err
	}
	m.streamReactors = append(m.streamReactors, r)
	return r, nil
}
