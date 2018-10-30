package module

type NetworkManager interface {
	Start()
	Stop()

	//CreateIfNotExists and return Membership
	GetMembership(name string) Membership
}

type Reactor interface {
	//goRoutine by Membership.receiveGoRoutine() like worker pattern
	//case broadcast, if return (true,nil) then rebroadcast when receiving
	OnReceive(subProtocol ProtocolInfo, bytes []byte, peerId PeerId) (bool, error)
	//call from Membership.onError() while message delivering
	OnError()
}

type Membership interface {
	RegistReactor(name string, reactor Reactor, subProtocols []ProtocolInfo) error

	//for Messaging, send packet to PeerToPeer.ch
	Broadcast(subProtocol ProtocolInfo, bytes []byte, broadcastType BroadcastType) error
	Multicast(subProtocol ProtocolInfo, bytes []byte, authority Authority) error
	Unicast(subProtocol ProtocolInfo, bytes []byte, peerId PeerId) error

	//for Authority management
	//권한,peerId 매핑정보는 다른 경로를 통해 공유되는 것을 전제로 한다.
	//TODO naming {authority, permission, privilege}
	//TODO naming {grant<>deny,allow<>disallow,add<>remove}
	AddRole(role Role, peerId PeerId) error
	RemoveRole(role Role, peerId PeerId) error
	HasRole(role Role, peerId PeerId) bool
	Roles(peerId PeerId) []Role
	GrantAuthority(authority Authority, role Role) error
	DenyAuthority(authority Authority, role Role) error
	HasAuthority(authority Authority, role Role) bool
	Authorities(role Role) []Authority
}

type BroadcastType string
type Role string
type Authority string

const (
	BROADCAST_ALL       BroadcastType = "all"
	BROADCAST_NEIGHBOR  BroadcastType = "neighbor"
	ROLE_VALIDATOR      Role          = "validator"
	AUTHORITY_BROADCAST Authority     = "broadcast"
)

type PeerId string

type ProtocolInfo interface {
	Id() byte
	Version() byte
}
