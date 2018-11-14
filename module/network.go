package module

import (
	"fmt"
)

type NetworkManager interface {
	//CreateIfNotExists and return Membership
	//if name == "" then return DefaultMembership with fixed PROTO_ID
	GetMembership(name string) Membership
}

type Reactor interface {
	//TODO call from Membership.workerRoutine
	//current direct call by Membership.onPacket from Peer.ReceiveRoutine
	//case broadcast, if return (true,nil) then rebroadcast when receiving
	OnReceive(subProtocol ProtocolInfo, bytes []byte, id PeerID) (bool, error)
	//TODO call from Membership.onError() while message delivering
	OnError()
}

type Membership interface {
	RegistReactor(name string, reactor Reactor, subProtocols []ProtocolInfo) error

	//for Messaging
	Broadcast(subProtocol ProtocolInfo, bytes []byte, broadcastType BroadcastType) error
	Multicast(subProtocol ProtocolInfo, bytes []byte, role Role) error
	Unicast(subProtocol ProtocolInfo, bytes []byte, id PeerID) error

	//for Authority management
	//Role,PeerID 매핑정보는 다른 경로를 통해 공유되는 것을 전제로 한다.
	//TODO naming {authority, permission, privilege}
	//TODO naming {grant<>deny,allow<>disallow,add<>remove}
	AddRole(role Role, peers ...PeerID)
	RemoveRole(role Role, peers ...PeerID)
	HasRole(role Role, id PeerID) bool
	Roles(id PeerID) []Role
	GrantAuthority(authority Authority, roles ...Role)
	DenyAuthority(authority Authority, roles ...Role)
	HasAuthority(authority Authority, role Role) bool
	Authorities(role Role) []Authority
}

type BroadcastType byte
type Role string
type Authority string

const (
	BROADCAST_ALL       BroadcastType = 0
	BROADCAST_NEIGHBOR  BroadcastType = 1
	ROLE_VALIDATOR      Role          = "validator"
	ROLE_SEED           Role          = "seed"
	AUTHORITY_BROADCAST Authority     = "broadcast"
)

type PeerID interface {
	Address
	Copy(b []byte)
}

type ProtocolInfo uint16

func NewProtocolInfo(id byte, version byte) ProtocolInfo {
	return ProtocolInfo(int(id)<<8 | int(version))
}
func (pi *ProtocolInfo) Id() byte {
	return byte(*pi >> 8)
}
func (pi *ProtocolInfo) Version() byte {
	return byte(*pi)
}
func (pi *ProtocolInfo) String() string {
	return fmt.Sprintf("{ID:%#02x,Ver:%#02x}", pi.Id(), pi.Version())
}
