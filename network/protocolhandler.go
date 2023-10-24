package network

import (
	"context"
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type protocolHandler struct {
	m            *manager
	protocol     module.ProtocolInfo
	subProtocols map[uint16]module.ProtocolInfo
	reactor      module.Reactor
	name         string
	priority     uint8
	policy       module.NotRegisteredProtocolPolicy
	receiveQueue Queue
	eventQueue   Queue
	//log
	logger log.Logger

	run chan bool
	mtx sync.RWMutex

	currentPkt *Packet
}

func newProtocolHandler(
	m *manager,
	pi module.ProtocolInfo,
	spiList []module.ProtocolInfo,
	r module.Reactor,
	name string,
	priority uint8,
	policy module.NotRegisteredProtocolPolicy,
	l log.Logger) *protocolHandler {
	phLogger := l.WithFields(log.Fields{LoggerFieldKeySubModule: name})
	ph := &protocolHandler{
		m:            m,
		protocol:     pi,
		subProtocols: make(map[uint16]module.ProtocolInfo),
		reactor:      r,
		name:         name,
		priority:     priority,
		policy:       policy,
		receiveQueue: NewQueue(DefaultReceiveQueueSize),
		eventQueue:   NewQueue(DefaultEventQueueSize),
		logger:       phLogger,
	}
	for _, sp := range spiList {
		k := sp.Uint16()
		if _, ok := ph.subProtocols[k]; ok {
			ph.logger.Infoln("newProtocolHandler", "already registered protocol", ph.name, ph.protocol, sp)
		}
		ph.subProtocols[k] = sp
	}

	ph.run = make(chan bool)

	go ph.receiveRoutine()
	go ph.eventRoutine()
	return ph
}

func (ph *protocolHandler) IsRun() bool {
	defer ph.mtx.RUnlock()
	ph.mtx.RLock()
	return ph.run != nil
}

func (ph *protocolHandler) Init() error {
	return nil
}

func (ph *protocolHandler) Term() {
	defer ph.mtx.Unlock()
	ph.mtx.Lock()
	if ph.run == nil {
		return
	}
	close(ph.run)
}

func (ph *protocolHandler) setReactor(r module.Reactor) {
	defer ph.mtx.Unlock()
	ph.mtx.Lock()

	ph.reactor = r
}

func (ph *protocolHandler) getReactor() module.Reactor {
	defer ph.mtx.RUnlock()
	ph.mtx.RLock()

	return ph.reactor
}

func (ph *protocolHandler) getPriority() uint8 {
	return ph.priority
}

func (ph *protocolHandler) getPolicy() module.NotRegisteredProtocolPolicy {
	return ph.policy
}

func (ph *protocolHandler) getName() string {
	return ph.name
}

func (ph *protocolHandler) getSubProtocol(spi module.ProtocolInfo) (module.ProtocolInfo, bool) {
	p, ok := ph.subProtocols[spi.Uint16()]
	return p, ok
}

func (ph *protocolHandler) getSubProtocols() []module.ProtocolInfo {
	spis := make([]module.ProtocolInfo, len(ph.subProtocols))
	i := 0
	for _, spi := range ph.subProtocols {
		spis[i] = spi
		i++
	}
	return spis
}

var ErrInProgress = errors.NewBase(errors.UnknownError, "InProgressError")

func (ph *protocolHandler) onPacketResult(pkt *Packet, isRelay bool, err error) {
	if isRelay && pkt.ttl == byte(module.BroadcastAll) && pkt.dest != p2pDestPeer {
		if err := ph.m.send(pkt); err != nil {
			ph.logger.Tracef("fail to relay error:{%+v} pkt=%s", err, pkt)
		}
	}
	if err != nil {
		ph.logger.Tracef("OnReceive returns err=%+v", err)
	}
}

func (ph *protocolHandler) HandleInBackground() (module.OnResult, error) {
	if ph.currentPkt == nil {
		return nil, errors.InvalidStateError.New("NotOnReceive()")
	}
	pkt := ph.currentPkt
	ph.currentPkt = nil

	return func(isRelay bool, err error) {
		ph.onPacketResult(pkt, isRelay, err)
	}, ErrInProgress
}

func (ph *protocolHandler) receiveRoutine() {
Loop:
	for {
		select {
		case <-ph.run:
			break Loop
		case <-ph.receiveQueue.Wait():
			for {
				ctx := ph.receiveQueue.Pop()
				if ctx == nil {
					break
				}
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
				p := ctx.Value(p2pContextKeyPeer).(*Peer)
				r := ph.getReactor()
				ph.currentPkt = pkt
				isRelay, err := r.OnReceive(pkt.subProtocol, pkt.payload, p.ID())
				if err != ErrInProgress || ph.currentPkt != nil {
					ph.currentPkt = nil
					ph.onPacketResult(pkt, isRelay, err)
				}
			}
		}
	}
}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (ph *protocolHandler) onPacket(pkt *Packet, p *Peer) {
	if !ph.IsRun() {
		return
	}

	_, ok := ph.getSubProtocol(pkt.subProtocol)
	if !ok {
		switch ph.policy {
		case module.NotRegisteredProtocolPolicyNone:
			ok = true
		case module.NotRegisteredProtocolPolicyDrop:
			ph.logger.Debugln("onPacket", "not registered protocol drop", ph.name, pkt.protocol, pkt.subProtocol, p.ID())
		case module.NotRegisteredProtocolPolicyClose:
			fallthrough
		default:
			p.CloseByError(ErrNotRegisteredProtocol)
			ph.logger.Infoln("onPacket", "not registered protocol", ph.name, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
	if ok {
		ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
		ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
		if ok = ph.receiveQueue.Push(ctx); !ok {
			ph.logger.Infoln("onPacket", "receiveQueue Push failure", ph.name, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
}

func (ph *protocolHandler) eventRoutine() {
Loop:
	for {
		select {
		case <-ph.run:
			break Loop
		case <-ph.eventQueue.Wait():
			for {
				ctx := ph.eventQueue.Pop()
				if ctx == nil {
					break
				}
				evt := ctx.Value(p2pContextKeyEvent).(string)
				p := ctx.Value(p2pContextKeyPeer).(*Peer)
				r := ph.getReactor()
				switch evt {
				case p2pEventJoin:
					r.OnJoin(p.ID())
				case p2pEventLeave:
					r.OnLeave(p.ID())
				case p2pEventDuplicate:
					ph.logger.Traceln("p2pEventDuplicate", p.ID())
				}
			}
		}
	}
}

func (ph *protocolHandler) onEvent(evt string, p *Peer) {
	if !ph.IsRun() {
		return
	}
	ph.logger.Traceln("onEvent", evt, p)
	ctx := context.WithValue(context.Background(), p2pContextKeyEvent, evt)
	ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
	if ok := ph.eventQueue.Push(ctx); !ok {
		ph.logger.Infoln("onEvent", "eventQueue Push failure", evt, p.ID())
	}
}

func (ph *protocolHandler) send(
	spi module.ProtocolInfo,
	b []byte,
	dest byte,
	ttl byte,
	forceSend bool,
	destPeer module.PeerID) error {
	if !ph.IsRun() {
		return ErrAlreadyClosed
	}

	if _, ok := ph.getSubProtocol(spi); !ok {
		return ErrNotRegisteredProtocol
	}

	if DefaultPacketPayloadMax < len(b) {
		return ErrIllegalArgument
	}
	pkt := NewPacket(ph.protocol, spi, b)
	pkt.priority = ph.getPriority()
	pkt.dest = dest
	pkt.ttl = ttl
	pkt.destPeer = destPeer
	pkt.forceSend = forceSend
	return ph.m.send(pkt)
}

func (ph *protocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	if err := ph.send(pi, b, p2pDestPeer, 1, true, id); err != nil {
		return newNetworkError(err, "unicast", id)
	}
	return nil
}

func (ph *protocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	if role >= module.RoleReserved {
		return newNetworkError(ErrNotRegisteredRole, "multicast", role)
	}
	if err := ph.send(pi, b, byte(role), 0, false, nil); err != nil {
		return newNetworkError(err, "multicast", role)
	}
	return nil
}

func (ph *protocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	if err := ph.send(pi, b, p2pDestAny, bt.TTL(), bt.ForceSend(), nil); err != nil {
		return newNetworkError(err, "broadcast", bt)
	}
	return nil
}

func (ph *protocolHandler) GetPeers() []module.PeerID {
	return ph.m.getPeersByProtocol(ph.protocol)
}
