package network

import (
	"fmt"

	"github.com/icon-project/goloop/module"
)

type membership struct {
	name             string
	protocol         module.ProtocolInfo
	p2p              *PeerToPeer
	roles            map[module.Role]*PeerIDSet
	authorities      map[module.Authority]*RoleSet
	reactors         map[string]module.Reactor
	onReceiveCbFuncs map[uint16]receiveCbFunc
	onErrorCbFuncs   map[uint16]func()
	subProtocols     map[uint16]module.ProtocolInfo
	destByRole       map[module.Role]byte
	//log
	log *logger
}

type receiveCbFunc func(pi module.ProtocolInfo, bytes []byte, id module.PeerID) (bool, error)

//type sendCbFunc func(pi module.ProtocolInfo, bytes []byte)
//Broadcast
//Multicast
//Unicast

func newMembership(name string, pi module.ProtocolInfo, p2p *PeerToPeer) *membership {
	ms := &membership{
		name:             name,
		protocol:         pi,
		p2p:              p2p,
		roles:            make(map[module.Role]*PeerIDSet),
		authorities:      make(map[module.Authority]*RoleSet),
		reactors:         make(map[string]module.Reactor),
		onReceiveCbFuncs: make(map[uint16]receiveCbFunc),
		onErrorCbFuncs:   make(map[uint16]func()),
		subProtocols:     make(map[uint16]module.ProtocolInfo),
		destByRole:       make(map[module.Role]byte),
		//
		log: newLogger("Membership", fmt.Sprintf("%s.%s.%s", p2p.channel, p2p.self.id, name)),
	}
	p2p.setCbFunc(pi, ms.onPacket, ms.onError)
	return ms
}

//TODO using worker pattern {pool or each packet or none} for reactor
func (ms *membership) workerRoutine() {

}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (ms *membership) onPacket(pkt *Packet, p *Peer) {
	ms.log.Println("onPacket", pkt)
	//TODO Check authority
	k := pkt.subProtocol.Uint16()
	if cbFunc := ms.onReceiveCbFuncs[k]; cbFunc != nil {
		pi := ms.subProtocols[k]
		r, err := cbFunc(pi, pkt.payload, p.ID())
		if err != nil {
			ms.log.Println(err)
		}
		if r {
			ms.log.Println("onPacket rebroadcast", pkt)
			ms.p2p.send(pkt)
		}
	}
}

func (ms *membership) onError(err error, p *Peer, pkt *Packet) {
	ms.log.Println("onError", err, p, pkt)
	if pkt != nil {
		k := pkt.subProtocol.Uint16()
		if cbFunc := ms.onErrorCbFuncs[k]; cbFunc != nil {
			cbFunc()
		}
	}
}

func (ms *membership) RegistReactor(name string, reactor module.Reactor, subProtocols []module.ProtocolInfo) error {
	if _, ok := ms.reactors[name]; ok {
		return ErrAlreadyRegisteredReactor
	}
	for _, sp := range subProtocols {
		k := sp.Uint16()
		if _, ok := ms.subProtocols[k]; ok {
			return ErrAlreadyRegisteredProtocol
		}
		ms.subProtocols[k] = sp
		ms.onReceiveCbFuncs[k] = reactor.OnReceive
		ms.onErrorCbFuncs[k] = reactor.OnError
		ms.log.Printf("RegistReactor.cbFuncs %#x %s", k, name)
	}
	return nil
}

func (ms *membership) Unicast(subProtocol module.ProtocolInfo, bytes []byte, id module.PeerID) error {
	p := ms.p2p.getPeer(id)
	if p == nil {
		return fmt.Errorf("not connected Peer %v", id)
	}

	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = p2pDestPeer
	ms.p2p.sendToPeer(pkt, p)
	return nil
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
