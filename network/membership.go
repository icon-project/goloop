package network

import (
	"github.com/icon-project/goloop/module"
)

type membership struct {
	name        string
	peerToPeer  PeerToPeer
	roles       map[module.Role][]module.PeerId
	authorities map[module.Authority][]module.Role
	reactors    []module.Reactor
}

//receive from PeerToPeer.receiveGoRoutine() using chan Packet
//using worker pattern {pool or each packet or none} for reactor
func (m *membership) receiveGoRoutine() {

}

//TODO naming {join<>leave, connect<>disconnect, add<>remove}
func (m *membership) onConnect(peer Peer) {

}

func (m *membership) onDisconnect(peer Peer) {

}

func (m *membership) RegistReactor(name string, reactor module.Reactor) {

}

func (m *membership) Unicast(subProtocol module.ProtocolInfo, bytes []byte, peerId module.PeerId) error {
	return nil
}

func (m *membership) Multicast(subProtocol module.ProtocolInfo, bytes []byte, authority module.Authority) error {
	return nil
}

func (m *membership) Broadcast(subProtocol module.ProtocolInfo, bytes []byte, broadcastType module.BroadcastType) error {
	return nil
}

func (m *membership) AddRole(role module.Role, peerId module.PeerId) error {
	return nil
}

func (m *membership) RemoveRole(role module.Role, peerId module.PeerId) error {
	return nil
}

func (m *membership) HasRole(role module.Role, peerId module.PeerId) bool {
	return false
}

func (m *membership) Roles(peerId module.PeerId) []module.Role {
	return nil
}

func (m *membership) GrantAuthority(authority module.Authority, role module.Role) error {
	return nil
}

func (m *membership) DenyAuthority(authority module.Authority, role module.Role) error {
	return nil
}

func (m *membership) HasAuthority(authority module.Authority, role module.Role) bool {
	return false
}

func (m *membership) Authorities(role module.Role) []module.Authority {
	return nil
}
