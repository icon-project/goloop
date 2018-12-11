package network

import (
	"context"
	"fmt"

	"github.com/icon-project/goloop/module"
)

type membership struct {
	name        string
	protocol    module.ProtocolInfo
	p2p         *PeerToPeer
	roles       map[module.Role]*PeerIDSet
	authorities map[module.Authority]*RoleSet
	reactors    map[string]*baseReactor
	protocolMap map[uint16]*baseReactor
	destByRole  map[module.Role]byte
	//log
	log *logger
}

type baseReactor struct {
	impl         module.Reactor
	name         string
	subProtocols map[uint16]module.ProtocolInfo
	receiveQueue *Queue
	eventQueue   *Queue
}

func newBaseReactor(name string, reactor module.Reactor, subProtocols []module.ProtocolInfo) *baseReactor {
	br := &baseReactor{
		impl:         reactor,
		name:         name,
		subProtocols: make(map[uint16]module.ProtocolInfo),
		receiveQueue: NewQueue(DefaultReceiveQueueSize),
		eventQueue:   NewQueue(DefaultEventQueueSize),
	}
	for _, sp := range subProtocols {
		k := sp.Uint16()
		br.subProtocols[k] = sp
	}
	return br
}

//call from Membership.onError() while message delivering
func (br *baseReactor) OnError(err error, subProtocol module.ProtocolInfo, bytes []byte, id module.PeerID) {

}

func newMembership(name string, pi module.ProtocolInfo, p2p *PeerToPeer) *membership {
	ms := &membership{
		name:        name,
		protocol:    pi,
		p2p:         p2p,
		roles:       make(map[module.Role]*PeerIDSet),
		authorities: make(map[module.Authority]*RoleSet),
		reactors:    make(map[string]*baseReactor),
		protocolMap: make(map[uint16]*baseReactor),
		destByRole:  make(map[module.Role]byte),
		//
		log: newLogger("Membership", fmt.Sprintf("%s.%s.%s", p2p.channel, p2p.self.id, name)),
	}
	p2p.setCbFunc(pi, ms.onPacket, ms.onError, ms.onEvent, p2pEventJoin, p2pEventLeave)
	return ms
}

//TODO using worker pattern {pool or each packet or none} for reactor
func (ms *membership) receiveRoutine(br *baseReactor) {
	for {
		<-br.receiveQueue.Wait()
		for {
			ctx := br.receiveQueue.Pop()
			if ctx == nil {
				break
			}
			pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
			p := ctx.Value(p2pContextKeyPeer).(*Peer)
			pi := br.subProtocols[pkt.subProtocol.Uint16()]
			// ms.log.Println("receiveRoutine", pi, p.ID)
			r, err := br.impl.OnReceive(pi, pkt.payload, p.ID())
			if err != nil {
				// ms.log.Println("receiveRoutine", err)
			}
			if r {
				if pkt.ttl == 1 {
					// ms.log.Println("receiveRoutine rebroadcast Ignore, not allowed when ttl=1", pkt)
				} else {
					// ms.log.Println("receiveRoutine rebroadcast", pkt)
					ms.p2p.send(pkt)
				}
			}
		}
	}
}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (ms *membership) onPacket(pkt *Packet, p *Peer) {
	// ms.log.Println("onPacket", pkt)
	//TODO Check authority
	k := pkt.subProtocol.Uint16()
	if br, ok := ms.protocolMap[k]; ok {
		ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
		ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
		if ok := br.receiveQueue.Push(ctx); !ok {
			// ms.log.Println("onPacket", "BaseReactor receiveQueue Push failure", br.name, pkt.protocol, pkt.subProtocol, p.ID())
		}
	}
}

func (ms *membership) onError(err error, p *Peer, pkt *Packet) {
	ms.log.Println("onError", err, p, pkt)
	if pkt != nil {
		k := pkt.subProtocol.Uint16()
		if br, ok := ms.protocolMap[k]; ok {
			pi := br.subProtocols[k]
			//TODO error notify
			br.OnError(err, pi, pkt.payload, p.ID())
		}
	}
}

func (ms *membership) eventRoutine(br *baseReactor) {
	for {
		<-br.eventQueue.Wait()
		for {
			ctx := br.eventQueue.Pop()
			if ctx == nil {
				break
			}
			evt := ctx.Value(p2pContextKeyEvent).(string)
			p := ctx.Value(p2pContextKeyPeer).(*Peer)
			ms.log.Println("eventRoutine", evt, p.ID())
			switch evt {
			case p2pEventJoin:
				// br.impl.OnJoin(p.ID())
			case p2pEventLeave:
				// br.impl.OnLeave(p.ID())
			}
		}
	}
}

func (ms *membership) onEvent(evt string, p *Peer) {
	// ms.log.Println("onEvent", evt, p)
	ctx := context.WithValue(context.Background(), p2pContextKeyEvent, evt)
	ctx = context.WithValue(ctx, p2pContextKeyPeer, p)
	for _, br := range ms.reactors {
		if ok := br.eventQueue.Push(ctx); !ok {
			// ms.log.Println("onEvent", "BaseReactor eventQueue Push failure", br.name, evt, p.ID())
		}
	}
}

func (ms *membership) RegistReactor(name string, reactor module.Reactor, subProtocols []module.ProtocolInfo) error {
	if _, ok := ms.reactors[name]; ok {
		return ErrAlreadyRegisteredReactor
	}
	br := newBaseReactor(name, reactor, subProtocols)
	for k := range br.subProtocols {
		if _, ok := ms.protocolMap[k]; ok {
			return ErrAlreadyRegisteredProtocol
		}
		ms.protocolMap[k] = br
		ms.log.Printf("RegistReactor.cbFuncs %#x %s", k, name)
	}
	ms.reactors[name] = br
	go ms.receiveRoutine(br)
	go ms.eventRoutine(br)
	return nil
}

func (ms *membership) Unicast(subProtocol module.ProtocolInfo, bytes []byte, id module.PeerID) error {
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = p2pDestPeer
	p := ms.p2p.getPeer(id, true)
	return ms.p2p.sendToPeer(pkt, p)
}

//TxMessage,PrevoteMessage, Send to Validators
func (ms *membership) Multicast(subProtocol module.ProtocolInfo, bytes []byte, role module.Role) error {
	if _, ok := ms.roles[role]; !ok {
		return ErrNotRegisteredRole
	}
	//TODO Check authority
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = ms.destByRole[role]
	return ms.p2p.send(pkt)
}

//ProposeMessage,PrecommitMessage,BlockMessage, Send to Citizen
func (ms *membership) Broadcast(subProtocol module.ProtocolInfo, bytes []byte, broadcastType module.BroadcastType) error {
	//TODO Check authority
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = p2pDestAny
	pkt.ttl = byte(broadcastType)
	return ms.p2p.send(pkt)
}

func (ms *membership) getRolePeerIDSet(role module.Role) *PeerIDSet {
	s, ok := ms.roles[role]
	if !ok {
		s := NewPeerIDSet()
		ms.roles[role] = s
		ms.destByRole[role] = byte(len(ms.roles) + p2pDestPeerGroup)
	}
	return s
}

func (ms *membership) SetRole(role module.Role, peers ...module.PeerID) {
	s := ms.getRolePeerIDSet(role)
	s.ClearAndAdd(peers...)
}

func (ms *membership) GetPeersByRole(role module.Role) []module.PeerID {
	s := ms.getRolePeerIDSet(role)
	return s.Array()
}

func (ms *membership) AddRole(role module.Role, peers ...module.PeerID) {
	s := ms.getRolePeerIDSet(role)
	for _, p := range peers {
		if !s.Contains(p) {
			s.Add(p)
		}
	}
}

func (ms *membership) RemoveRole(role module.Role, peers ...module.PeerID) {
	s := ms.getRolePeerIDSet(role)
	for _, p := range peers {
		s.Remove(p)
	}
}

func (ms *membership) HasRole(role module.Role, id module.PeerID) bool {
	s := ms.getRolePeerIDSet(role)
	return s.Contains(id)
}

func (ms *membership) Roles(id module.PeerID) []module.Role {
	var i int
	s := make([]module.Role, 0, len(ms.roles))
	for k, v := range ms.roles {
		if v.Contains(id) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}

func (ms *membership) getAuthorityRoleSet(authority module.Authority) *RoleSet {
	s, ok := ms.authorities[authority]
	if !ok {
		s := NewRoleSet()
		ms.authorities[authority] = s
	}
	return s
}

func (ms *membership) GrantAuthority(authority module.Authority, roles ...module.Role) {
	s := ms.getAuthorityRoleSet(authority)
	for _, r := range roles {
		if !s.Contains(r) {
			s.Add(r)
		}
	}
}

func (ms *membership) DenyAuthority(authority module.Authority, roles ...module.Role) {
	l := ms.getAuthorityRoleSet(authority)
	for _, r := range roles {
		l.Remove(r)
	}
}

func (ms *membership) HasAuthority(authority module.Authority, role module.Role) bool {
	l := ms.getAuthorityRoleSet(authority)
	return l.Contains(role)
}

func (ms *membership) Authorities(role module.Role) []module.Authority {
	var i int
	s := make([]module.Authority, len(ms.authorities))
	for k, v := range ms.authorities {
		if v.Contains(role) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}
