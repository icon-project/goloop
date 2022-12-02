package network

import (
	"context"
	"sync"

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
				isRelay, _ := r.OnReceive(pkt.subProtocol, pkt.payload, p.ID())
				if isRelay && pkt.ttl == byte(module.BROADCAST_ALL) && pkt.dest != p2pDestPeer {
					if err := ph.m.relay(pkt); err != nil {
						ph.logger.Tracef("fail to relay error:{%+v} pkt:%s", err, pkt)
					}
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

func (ph *protocolHandler) Unicast(spi module.ProtocolInfo, b []byte, id module.PeerID) error {
	if !ph.IsRun() {
		return NewUnicastError(ErrAlreadyClosed, id)
	}

	if _, ok := ph.getSubProtocol(spi); !ok {
		return NewUnicastError(ErrNotRegisteredProtocol, id)
	}

	ph.logger.Traceln("Unicast", spi, len(b), id)
	if err := ph.m.unicast(ph.protocol, spi, b, id); err != nil {
		return NewUnicastError(err, id)
	}
	return nil
}

//TxMessage,PrevoteMessage, Send to Validators
func (ph *protocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	if !ph.IsRun() {
		return NewMulticastError(ErrAlreadyClosed, role)
	}
	spi := module.ProtocolInfo(pi.Uint16())
	if _, ok := ph.getSubProtocol(spi); !ok {
		return NewMulticastError(ErrNotRegisteredProtocol, role)
	}

	ph.logger.Traceln("Multicast", pi, len(b), role)
	if err := ph.m.multicast(ph.protocol, spi, b, role); err != nil {
		return NewMulticastError(err, role)
	}
	return nil
}

//ProposeMessage,PrecommitMessage,BlockMessage, Send to Citizen
func (ph *protocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	if !ph.IsRun() {
		return NewBroadcastError(ErrAlreadyClosed, bt)
	}
	spi := module.ProtocolInfo(pi.Uint16())
	if _, ok := ph.getSubProtocol(spi); !ok {
		return NewBroadcastError(ErrNotRegisteredProtocol, bt)
	}

	ph.logger.Traceln("Broadcast", pi, len(b), bt)
	if err := ph.m.broadcast(ph.protocol, spi, b, bt); err != nil {
		return NewBroadcastError(err, bt)
	}
	return nil
}

func (ph *protocolHandler) GetPeers() []module.PeerID {
	return ph.m.getPeersByProtocol(ph.protocol)
}
