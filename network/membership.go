package network

import (
	"container/list"
	"fmt"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

type membership struct {
	name        string
	protocol    module.ProtocolInfo
	p2p         *PeerToPeer
	roles       map[module.Role]*PeerIdList
	authorities map[module.Authority]*RoleList
	reactors    map[string]module.Reactor
	cbFuncs     map[module.ProtocolInfo]receiveCbFunc
	destByRole  map[module.Role]byte
}

type receiveCbFunc func(pi module.ProtocolInfo, bytes []byte, id module.PeerID) (bool, error)

func newMembership(name string, pi module.ProtocolInfo, p2p *PeerToPeer) *membership {
	ms := &membership{
		name:        name,
		protocol:    pi,
		p2p:         p2p,
		roles:       make(map[module.Role]*PeerIdList),
		authorities: make(map[module.Authority]*RoleList),
		reactors:    make(map[string]module.Reactor),
		cbFuncs:     make(map[module.ProtocolInfo]receiveCbFunc),
		destByRole:  make(map[module.Role]byte),
	}
	p2p.setPacketCbFunc(pi, ms.onPacket)
	return ms
}

//TODO using worker pattern {pool or each packet or none} for reactor
func (ms *membership) workerRoutine() {

}

//callback from PeerToPeer.onPacket() in Peer.onReceiveRoutine
func (ms *membership) onPacket(pkt *Packet, p *Peer) {
	log.Println("Membership.onPacket", pkt)
	//Check authority
	//roles := Roles(pkt.src)
	//auth := Authority(pkt.cast)
	//r := HasAuthority(auth, role) range roles
	//if r == true

	if cbFunc := ms.cbFuncs[pkt.subProtocol]; cbFunc != nil {
		r, err := cbFunc(pkt.subProtocol, pkt.payload, p.ID())
		if err != nil {
			log.Println(err)
		}
		if r {
			log.Println("Membership.onPacket rebroadcast", pkt)
			ms.p2p.ch <- pkt
		}
	}
}

func (ms *membership) RegistReactor(name string, reactor module.Reactor, subProtocols []module.ProtocolInfo) error {
	if _, ok := ms.reactors[name]; ok {
		return common.ErrIllegalArgument
	}
	for _, sp := range subProtocols {
		if _, ok := ms.cbFuncs[sp]; ok {
			return common.ErrIllegalArgument
		}
		ms.cbFuncs[sp] = reactor.OnReceive
		log.Printf("Membership.RegistReactor.cbFuncs %#x %s", sp.Uint16(), name)
	}
	return nil
}

func (ms *membership) Unicast(subProtocol module.ProtocolInfo, bytes []byte, id module.PeerID) error {
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = p2pDestPeer
	ms.p2p.sendToPeer(pkt, id)
	return nil
}

//TxMessage,VoteMessage, Send to Validators
func (ms *membership) Multicast(subProtocol module.ProtocolInfo, bytes []byte, role module.Role) error {
	if _, ok := ms.roles[role]; !ok {
		return common.ErrIllegalArgument
	}

	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = ms.destByRole[role]
	ms.p2p.ch <- pkt
	return nil
}

//ProposeMessage,BlockMessage, Send to Citizen
func (ms *membership) Broadcast(subProtocol module.ProtocolInfo, bytes []byte, broadcastType module.BroadcastType) error {
	pkt := NewPacket(subProtocol, bytes)
	pkt.protocol = ms.protocol
	pkt.dest = p2pDestAny
	pkt.ttl = byte(broadcastType)
	ms.p2p.ch <- pkt
	return nil
}

func (ms *membership) getRolePeerIDList(role module.Role) *PeerIdList {
	l, ok := ms.roles[role]
	if !ok {
		l := NewPeerIdList()
		ms.roles[role] = l
		ms.destByRole[role] = byte(len(ms.roles) + p2pDestPeerGroup)
	}
	return l
}

func (ms *membership) AddRole(role module.Role, peers ...module.PeerID) {
	l := ms.getRolePeerIDList(role)
	for _, p := range peers {
		if !l.Has(p) {
			l.PushBack(p)
		}
	}
}

func (ms *membership) RemoveRole(role module.Role, peers ...module.PeerID) {
	l := ms.getRolePeerIDList(role)
	for _, p := range peers {
		l.Remove(p)
	}
}

func (ms *membership) HasRole(role module.Role, id module.PeerID) bool {
	l := ms.getRolePeerIDList(role)
	return l.Has(id)
}

func (ms *membership) Roles(id module.PeerID) []module.Role {
	var i int
	s := make([]module.Role, 0, len(ms.roles))
	for k, v := range ms.roles {
		if v.Has(id) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}

func (ms *membership) getAuthorityRoleList(authority module.Authority) *RoleList {
	l, ok := ms.authorities[authority]
	if !ok {
		l := NewRoleList()
		ms.authorities[authority] = l
	}
	return l
}

func (ms *membership) GrantAuthority(authority module.Authority, roles ...module.Role) {
	l := ms.getAuthorityRoleList(authority)
	for _, r := range roles {
		if !l.Has(r) {
			l.PushBack(r)
		}
	}
}

func (ms *membership) DenyAuthority(authority module.Authority, roles ...module.Role) {
	l := ms.getAuthorityRoleList(authority)
	for _, r := range roles {
		l.Remove(r)
	}
}

func (ms *membership) HasAuthority(authority module.Authority, role module.Role) bool {
	l := ms.getAuthorityRoleList(authority)
	return l.Has(role)
}

func (ms *membership) Authorities(role module.Role) []module.Authority {
	var i int
	s := make([]module.Authority, len(ms.authorities))
	for k, v := range ms.authorities {
		if v.Has(role) {
			s = append(s, k)
			i++
		}
	}
	return s[:i]
}

type PeerIdList struct {
	*list.List
}

func NewPeerIdList() *PeerIdList {
	return &PeerIdList{list.New()}
}

func (l *PeerIdList) get(v module.PeerID) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(module.PeerID); s.Equal(v) {
			return e
		}
	}
	return nil
}

func (l *PeerIdList) Remove(v module.PeerID) bool {
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *PeerIdList) Has(v module.PeerID) bool {
	return l.get(v) != nil
}

func (l *PeerIdList) IsEmpty() bool {
	return l.Len() == 0
}

func (l *PeerIdList) String() string {
	s := make([]string, 0, l.Len())
	for e := l.Front(); e != nil; e = e.Next() {
		s = append(s, e.Value.(module.PeerID).String())
	}
	return fmt.Sprintf("%v", s)
}

type RoleList struct {
	*list.List
}

func NewRoleList() *RoleList {
	return &RoleList{list.New()}
}

func (l *RoleList) get(v module.Role) *list.Element {
	for e := l.Front(); e != nil; e = e.Next() {
		if s := e.Value.(module.Role); s == v {
			return e
		}
	}
	return nil
}

func (l *RoleList) Remove(v module.Role) bool {
	if e := l.get(v); e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

func (l *RoleList) Has(v module.Role) bool {
	return l.get(v) != nil
}
