package module

type NetworkManager interface {
	//CreateIfNotExists and return Membership
	//if name == "" then return DefaultMembership with fixed PROTO_ID
	GetMembership(name string) Membership
	GetPeers() []PeerID
}

type Reactor interface {
	//case broadcast and multicast, if return (true,nil) then rebroadcast
	OnReceive(subProtocol ProtocolInfo, bytes []byte, id PeerID) (bool, error)
	//TODO call from Membership.onError() while message delivering
	OnError()
	OnJoin(id PeerID)
	OnLeave(id PeerID)
}

type Membership interface {
	RegistReactor(name string, reactor Reactor, subProtocols []ProtocolInfo) error

	//for Messaging
	Broadcast(subProtocol ProtocolInfo, bytes []byte, broadcastType BroadcastType) error
	Multicast(subProtocol ProtocolInfo, bytes []byte, role Role) error
	Unicast(subProtocol ProtocolInfo, bytes []byte, id PeerID) error

	//for Authority management
	SetRole(role Role, peers ...PeerID)
	GetPeersByRole(role Role) []PeerID
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

type ProtocolInfo interface {
	ID() byte
	Version() byte
	String() string
	Copy(b []byte)
	Uint16() uint16
}

type NetworkTransport interface {
	Listen() error
	Close() error
	Dial(address string, channel string) error
	PeerID() PeerID
	Address() string
}
