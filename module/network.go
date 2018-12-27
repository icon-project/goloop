package module

type NetworkManager interface {
	GetPeers() []PeerID

	RegisterReactor(name string, reactor Reactor, piList []ProtocolInfo, priority uint8) (ProtocolHandler, error)

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

type Reactor interface {
	//case broadcast and multicast, if return (true,nil) then rebroadcast
	OnReceive(pi ProtocolInfo, b []byte, id PeerID) (bool, error)
	OnError(err error, pi ProtocolInfo, b []byte, id PeerID)
	OnJoin(id PeerID)
	OnLeave(id PeerID)
}

type ProtocolHandler interface {
	//for Messaging
	Broadcast(pi ProtocolInfo, b []byte, bt BroadcastType) error
	Multicast(pi ProtocolInfo, b []byte, role Role) error
	Unicast(pi ProtocolInfo, b []byte, id PeerID) error
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

type NetworkError interface {
	error
	Temporary() bool // Is the error temporary?
}
