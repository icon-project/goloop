package network

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	clock "github.com/icon-project/goloop/test/clock"
)

type tReactorItem struct {
	name     string
	reactor  module.Reactor
	piList   []module.ProtocolInfo
	priority uint8
}

type tPacket struct {
	pi module.ProtocolInfo
	b  []byte
	id module.PeerID
}

var tNMMu sync.Mutex

type tNetworkManager struct {
	// immutable
	module.NetworkManager
	id module.PeerID

	// mutable
	reactorItems []*tReactorItem
	peers        []*tNetworkManager
	drop         bool
	recvBuf      []*tPacket
}

type tProtocolHandler struct {
	nm *tNetworkManager
	ri *tReactorItem
}

func newTNetworkManager(id module.PeerID) *tNetworkManager {
	return &tNetworkManager{id: id}
}

func (nm *tNetworkManager) GetPeers() []module.PeerID {
	tNMMu.Lock()
	defer tNMMu.Unlock()

	res := make([]module.PeerID, len(nm.peers))
	for i := range nm.peers {
		res[i] = nm.peers[i].id
	}
	return res
}

func (nm *tNetworkManager) RegisterReactor(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	tNMMu.Lock()
	defer tNMMu.Unlock()

	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
}

func (nm *tNetworkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	return registerReactorForStreams(nm, name, pi, reactor, piList, priority, policy, &common.GoTimeClock{})
}

func (nm *tNetworkManager) join(nm2 *tNetworkManager) {
	tNMMu.Lock()
	nm.peers = append(nm.peers, nm2)
	nm2.peers = append(nm2.peers, nm)
	reactorItems := append([]*tReactorItem(nil), nm.reactorItems...)
	reactorItems2 := append([]*tReactorItem(nil), nm2.reactorItems...)
	tNMMu.Unlock()

	for _, r := range reactorItems {
		r.reactor.OnJoin(nm2.id)
	}
	for _, r := range reactorItems2 {
		r.reactor.OnJoin(nm.id)
	}
}

func (nm *tNetworkManager) onReceiveUnicast(pi module.ProtocolInfo, b []byte, from module.PeerID) {
	tNMMu.Lock()
	defer tNMMu.Unlock()
	nm.recvBuf = append(nm.recvBuf, &tPacket{pi, b, from})
}

func (nm *tNetworkManager) processRecvBuf() {
	tNMMu.Lock()
	recvBuf := append([]*tPacket(nil), nm.recvBuf...)
	reactorItems := append([]*tReactorItem(nil), nm.reactorItems...)
	tNMMu.Unlock()
	for _, p := range recvBuf {
		for _, r := range reactorItems {
			r.reactor.OnReceive(p.pi, p.b, p.id)
		}
	}
	nm.recvBuf = nil
}

func (ph *tProtocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		if p.id.Equal(id) {
			p.onReceiveUnicast(pi, b, ph.nm.id)
			return nil
		}
	}
	return errors.Errorf("Unknown peer")
}

func (ph *tProtocolHandler) GetPeers() []module.PeerID {
	return ph.nm.GetPeers()
}

func createAPeerID() module.PeerID {
	return NewPeerIDFromAddress(wallet.New().Address())
}

type tReceiveEvent struct {
	PI module.ProtocolInfo
	B  []byte
	ID module.PeerID
}

type tReceiveStreamMessageEvent struct {
	PI module.ProtocolInfo
	B  []byte
	ID module.PeerID
	SM streamMessage
}

type tJoinEvent struct {
	ID module.PeerID
}

type tLeaveEvent struct {
	ID module.PeerID
}

type tReactor struct {
	useStreamMessageEvent bool
	ch                    chan interface{}
}

func newTReactor() *tReactor {
	return &tReactor{ch: make(chan interface{}, 5)}
}

func (r *tReactor) OnReceive(pi module.ProtocolInfo, b []byte, id module.PeerID) (bool, error) {
	if r.useStreamMessageEvent {
		sm := &streamMessage{}
		codec.UnmarshalFromBytes(b, sm)
		r.ch <- tReceiveStreamMessageEvent{pi, b, id, *sm}
	} else {
		r.ch <- tReceiveEvent{pi, b, id}
	}
	return false, nil
}

func (r *tReactor) OnJoin(id module.PeerID) {
	r.ch <- tJoinEvent{id}
}

func (r *tReactor) OnLeave(id module.PeerID) {
	r.ch <- tLeaveEvent{id}
}

const (
	pi0 module.ProtocolInfo = iota
	pi1
)

var pis = []module.ProtocolInfo{pi0, pi1}

type streamTestSetUp struct {
	nm *tNetworkManager
	r  *tReactor
	ph module.ProtocolHandler

	// for non stream
	nm2 *tNetworkManager
	r2  *tReactor
	ph2 module.ProtocolHandler

	clock    *clock.Clock
	payloads [][]byte
	tick     time.Duration
}

func newStreamTestSetUp(t *testing.T) *streamTestSetUp {
	s := &streamTestSetUp{}
	s.clock = &clock.Clock{}
	s.nm = newTNetworkManager(createAPeerID())
	s.nm2 = newTNetworkManager(createAPeerID())
	s.nm.join(s.nm2)
	s.r = newTReactor()
	var err error
	s.ph, err = registerReactorForStreams(s.nm, "reactorA", 0, s.r, pis, 1, module.NotRegisteredProtocolPolicyClose, s.clock)
	assert.Nil(t, err)
	s.r2 = newTReactor()
	s.ph2, err = s.nm2.RegisterReactor("reactorA", 0, s.r2, pis, 1, module.NotRegisteredProtocolPolicyClose)
	assert.Nil(t, err)
	s.r2.useStreamMessageEvent = true

	const NUM_PAYLOADS = 10
	for i := 0; i < NUM_PAYLOADS; i++ {
		s.payloads = append(s.payloads, []byte{byte(i + 1)})
	}
	s.tick = configPeerAckTimeout / 10
	return s
}

func unicastStreamMessage(ph module.ProtocolHandler, pi module.ProtocolInfo, seq uint16, ack uint16, payload []byte, id module.PeerID) {
	b := codec.MustMarshalToBytes(&streamMessage{seq, ack, payload})
	ph.Unicast(pi, b, id)
}

func repeat(t *testing.T, s uint16, e uint16, test func(*testing.T, uint16)) {
	for i := s; i != e; i++ {
		test(t, i)
	}
}

func assertMessageReceived(t *testing.T, ch <-chan interface{}, pi module.ProtocolInfo, b []byte, id module.PeerID) {
	select {
	case ev := <-ch:
		assert.Equal(t, tReceiveEvent{pi, b, id}, ev.(tReceiveEvent))
	default:
		assert.Fail(t, "retransmission is expected")
	}
}

func assertNoEvent(t *testing.T, ch <-chan interface{}) {
	select {
	case ev := <-ch:
		assert.Fail(t, fmt.Sprintf("unexpected message %+v", ev))
	default:
	}
}

func assertStreamMessageReceived(t *testing.T, ch <-chan interface{}, seq uint16, ack uint16, payload []byte) {
	select {
	case ev := <-ch:
		assert.Equal(t, streamMessage{seq, ack, payload}, ev.(tReceiveStreamMessageEvent).SM)
	default:
		assert.Fail(t, "message is expected")
	}
}

func TestStream_SendAndReceive(t *testing.T) {
	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	nm.join(nm2)
	ph, err := nm.RegisterReactorForStreams("reactorA", 0, newTReactor(), pis, 1, module.NotRegisteredProtocolPolicyClose)
	assert.Nil(t, err)
	r := newTReactor()
	_, err = nm2.RegisterReactorForStreams("reactorA", 0, r, pis, 1, module.NotRegisteredProtocolPolicyClose)
	assert.Nil(t, err)
	payload := []byte{0, 1}
	ph.Unicast(pi0, payload, nm2.id)
	nm2.processRecvBuf()
	ev := <-r.ch
	assert.Equal(t, tReceiveEvent{pi0, payload, nm.id}, ev)
}

func TestStream_SendAndReceiveComplex(t *testing.T) {
	clock := &clock.Clock{}
	nm := newTNetworkManager(createAPeerID())
	nm2 := newTNetworkManager(createAPeerID())
	nm.join(nm2)
	ph, err := registerReactorForStreams(nm, "reactorA", 0, newTReactor(), pis, 1, module.NotRegisteredProtocolPolicyClose, clock)
	assert.Nil(t, err)
	r := newTReactor()
	_, err = registerReactorForStreams(nm2, "reactorA", 0, r, pis, 1, module.NotRegisteredProtocolPolicyClose, clock)
	assert.Nil(t, err)

	const ITER = 100000
	go func() {
		for i := 0; i < ITER; i++ {
			payload := []byte{uint8(i), uint8(i >> 8)}
			nm.drop = (i%3 == 0)
			ph.Unicast(pi0, payload, nm2.id)
			nm.processRecvBuf()
			nm2.processRecvBuf()
			clock.PassTime(configPeerAckTimeout / 10)
			nm.processRecvBuf()
			nm2.processRecvBuf()
		}
		nm.drop = false
		for i := 0; i < 10; i++ {
			clock.PassTime(configPeerAckTimeout)
			nm.processRecvBuf()
			nm2.processRecvBuf()
		}
	}()
	for i := 0; i < ITER; i++ {
		payload := []byte{uint8(i), uint8(i >> 8)}
		ev := <-r.ch
		assert.Equal(t, tReceiveEvent{pi0, payload, nm.id}, ev)
	}
}

func TestStream_NoRepostOnTimelyAck(t *testing.T) {
	repeat(t, 0xFFFF-1, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)

		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		unicastStreamMessage(s.ph2, pi0, 0, base+1, nil, s.nm.id)
		s.nm.processRecvBuf()
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_RepostOnAckTimeout(t *testing.T) {
	repeat(t, 0xFFFF-1, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)

		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_RepostSingleMessageOnAckTimeout(t *testing.T) {
	repeat(t, 0xFFFF-2, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)
		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)
		s.ph.Unicast(pi0, s.payloads[1], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+2, 0, s.payloads[1])
		assertNoEvent(t, s.r2.ch)
		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)
		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_PauseMessagePostingOnAckTimeout(t *testing.T) {
	repeat(t, 0xFFFF, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		s.ph.Unicast(pi0, s.payloads[1], s.nm2.id)
		s.nm2.processRecvBuf()
		assertNoEvent(t, s.r2.ch)

		unicastStreamMessage(s.ph2, pi0, 0, base+1, nil, s.nm.id)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+2, 0, s.payloads[1])
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_RetransmissionComplex(t *testing.T) {
	repeat(t, 0xFFFF-5, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(s.tick)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		s.ph.Unicast(pi0, s.payloads[1], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+2, 0, s.payloads[1])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(s.tick)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		s.ph.Unicast(pi0, s.payloads[2], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+3, 0, s.payloads[2])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(s.tick)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		s.ph.Unicast(pi0, s.payloads[3], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+4, 0, s.payloads[3])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(s.tick)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		s.ph.Unicast(pi0, s.payloads[4], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+5, 0, s.payloads[4])
		assertNoEvent(t, s.r2.ch)

		unicastStreamMessage(s.ph2, pi0, 0, base+2, nil, s.nm.id)
		s.nm.processRecvBuf()
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+3, 0, s.payloads[2])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+3, 0, s.payloads[2])
		assertNoEvent(t, s.r2.ch)

		unicastStreamMessage(s.ph2, pi0, 0, base+3, nil, s.nm.id)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+4, 0, s.payloads[3])
		assertStreamMessageReceived(t, s.r2.ch, base+5, 0, s.payloads[4])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+4, 0, s.payloads[3])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+4, 0, s.payloads[3])
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_RetransmitAllPendingOnAckAfterTimeout(t *testing.T) {
	repeat(t, 0xFFFF-2, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(s.tick * 5)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		s.ph.Unicast(pi0, s.payloads[1], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+2, 0, s.payloads[1])
		assertNoEvent(t, s.r2.ch)

		s.clock.PassTime(s.tick * 5)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])
		assertNoEvent(t, s.r2.ch)

		unicastStreamMessage(s.ph2, pi0, 0, base+1, nil, s.nm.id)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+2, 0, s.payloads[1])
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_DoNothingOnFutureAck(t *testing.T) {
	repeat(t, 0xFFFF-1, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])

		unicastStreamMessage(s.ph2, pi0, 0, base+2, nil, s.nm.id)
		s.nm.processRecvBuf()
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_DoNothingOnPastAck(t *testing.T) {
	repeat(t, 0xFFFF-1, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setSeqByForce(base)

		s.ph.Unicast(pi0, s.payloads[0], s.nm2.id)
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, base+1, 0, s.payloads[0])

		unicastStreamMessage(s.ph2, pi0, 0, base+0, nil, s.nm.id)
		s.nm.processRecvBuf()
		assertNoEvent(t, s.r2.ch)
	})
}

func TestStream_Reordering(t *testing.T) {
	repeat(t, 0xFFFF-1, 0, func(t *testing.T, base uint16) {
		s := newStreamTestSetUp(t)
		s.ph.(*streamReactor).streamForPeer(s.nm2.id).setPeerSeqByForce(base)

		unicastStreamMessage(s.ph2, pi0, base+1, 0, s.payloads[0], s.nm.id)
		s.nm.processRecvBuf()
		assertMessageReceived(t, s.r.ch, pi0, s.payloads[0], s.nm2.id)
		assertNoEvent(t, s.r.ch)

		unicastStreamMessage(s.ph2, pi0, base+3, 0, s.payloads[2], s.nm.id)
		s.nm.processRecvBuf()
		assertNoEvent(t, s.r.ch)

		s.clock.PassTime(configPeerAckTimeout)
		s.nm.processRecvBuf()
		s.nm2.processRecvBuf()
		assertStreamMessageReceived(t, s.r2.ch, 0, base+1, nil)
		assertNoEvent(t, s.r.ch)

		unicastStreamMessage(s.ph2, pi0, base+2, 0, s.payloads[1], s.nm.id)
		s.nm.processRecvBuf()
		assertMessageReceived(t, s.r.ch, pi0, s.payloads[1], s.nm2.id)
		assertNoEvent(t, s.r.ch)

		unicastStreamMessage(s.ph2, pi0, base+3, 0, s.payloads[2], s.nm.id)
		s.nm.processRecvBuf()
		assertMessageReceived(t, s.r.ch, pi0, s.payloads[2], s.nm2.id)
		assertNoEvent(t, s.r.ch)
	})
}
