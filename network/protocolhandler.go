package network

import (
	"context"
	"fmt"

	"github.com/icon-project/goloop/module"
)

type protocolHandler struct {
	m            *manager
	protocol     protocolInfo
	subProtocols map[protocolInfo]module.ProtocolInfo
	reactor      module.Reactor
	name         string
	priority     uint8
	receiveQueue *Queue
	eventQueue   *Queue
	//log
	log *logger
}

func newProtocolHandler(m *manager, pi protocolInfo, spiList []module.ProtocolInfo, r module.Reactor, name string, priority uint8) *protocolHandler {
	ph := &protocolHandler{
		m:            m,
		protocol:     pi,
		subProtocols: make(map[protocolInfo]module.ProtocolInfo),
		reactor:      r,
		name:         name,
		priority:     priority,
		receiveQueue: NewQueue(DefaultReceiveQueueSize),
		eventQueue:   NewQueue(DefaultEventQueueSize),
		log:          newLogger("ProtocolHandler", fmt.Sprintf("%s.%s.%s", m.Channel(), m.PeerID(), name)),
	}
	for _, sp := range spiList {
		k := protocolInfo(sp.Uint16())
		ph.subProtocols[k] = sp
	}
	go ph.receiveRoutine()
	go ph.eventRoutine()
	return ph
}

//TODO using worker pattern {pool or each packet or none} for reactor
func (ph *protocolHandler) receiveRoutine() {
	for {
		<-ph.receiveQueue.Wait()
		for {
			ctx := ph.receiveQueue.Pop()
			if ctx == nil {
				break
			}
			pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
			p := ctx.Value(p2pContextKeyPeer).(*Peer)
			pi := ph.subProtocols[pkt.subProtocol]
			// ph.log.Println("receiveRoutine", pi, p.ID)
			r, err := ph.reactor.OnReceive(pi, pkt.payload, p.ID())
			if err != nil {
				// ph.log.Println("receiveRoutine", err)
			}

			if r && pkt.ttl != 1 {
				if err := ph.m.relay(pkt); err != nil {
					ph.log.Println("Warning", "receiveRoutine", "relay", err)
				}
			}
		}
	}
}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (ph *protocolHandler) onPacket(pkt *Packet, p *Peer) {
	// ph.log.Println("onPacket", pkt)
	//TODO Check authority
	k := pkt.subProtocol
	if _, ok := ph.subProtocols[k]; ok {
		ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
		ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
		if ok := ph.receiveQueue.Push(ctx); !ok {
			// ph.log.Println("onPacket", "receiveQueue Push failure", ph.name, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
}

func (ph *protocolHandler) onError(err error, p *Peer, pkt *Packet) {
	ph.log.Println("onError", err, p, pkt)
	if err != nil {
		k := pkt.subProtocol
		if pi, ok := ph.subProtocols[k]; ok {
			var b []byte = nil
			var id module.PeerID = nil
			if pkt != nil {
				b = pkt.payload
			}
			if p != nil {
				id = p.ID()
			}
			ph.reactor.OnError(err, pi, b, id)
		}
	}
}

func (ph *protocolHandler) eventRoutine() {
	for {
		<-ph.eventQueue.Wait()
		for {
			ctx := ph.eventQueue.Pop()
			if ctx == nil {
				break
			}
			evt := ctx.Value(p2pContextKeyEvent).(string)
			p := ctx.Value(p2pContextKeyPeer).(*Peer)
			ph.log.Println("eventRoutine", evt, p.ID())
			switch evt {
			case p2pEventJoin:
				ph.reactor.OnJoin(p.ID())
			case p2pEventLeave:
				ph.reactor.OnLeave(p.ID())
			case p2pEventDuplicate:
				ph.reactor.OnLeave(p.ID())
			}
		}
	}
}

//TODO check case p2p.onEvent
func (ph *protocolHandler) onEvent(evt string, p *Peer) {
	// ph.log.Println("onEvent", evt, p)
	ctx := context.WithValue(context.Background(), p2pContextKeyEvent, evt)
	ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
	if ok := ph.eventQueue.Push(ctx); !ok {
		// ph.log.Println("onEvent", "eventQueue Push failure", evt, p.ID())
	}
}

func (ph *protocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	spi := protocolInfo(pi.Uint16())
	if _, ok := ph.subProtocols[spi]; !ok {
		return ErrNotRegisteredProtocol
	}
	return ph.m.unicast(ph.protocol, spi, b, id)
}

//TxMessage,PrevoteMessage, Send to Validators
func (ph *protocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	spi := protocolInfo(pi.Uint16())
	if _, ok := ph.subProtocols[spi]; !ok {
		return ErrNotRegisteredProtocol
	}
	return ph.m.multicast(ph.protocol, spi, b, role)
}

//ProposeMessage,PrecommitMessage,BlockMessage, Send to Citizen
func (ph *protocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	spi := protocolInfo(pi.Uint16())
	if _, ok := ph.subProtocols[spi]; !ok {
		return ErrNotRegisteredProtocol
	}
	return ph.m.broadcast(ph.protocol, spi, b, bt)
}
