package module

type NetworkManager interface {
	Start() error
	Stop() error

	GetPeers() []PeerID

	RegisterReactor(name string, reactor Reactor, piList []ProtocolInfo, priority uint8) (ProtocolHandler, error)
	RegisterReactorForStreams(name string, reactor Reactor, piList []ProtocolInfo, priority uint8) (ProtocolHandler, error)

	SetRole(role Role, peers ...PeerID)
	GetPeersByRole(role Role) []PeerID
	AddRole(role Role, peers ...PeerID)
	RemoveRole(role Role, peers ...PeerID)
	HasRole(role Role, id PeerID) bool
	Roles(id PeerID) []Role
}

type Reactor interface {
	//case broadcast and multicast, if return (true,nil) then rebroadcast
	OnReceive(pi ProtocolInfo, b []byte, id PeerID) (bool, error)
	OnFailure(err error, pi ProtocolInfo, b []byte)
	OnJoin(id PeerID)
	OnLeave(id PeerID)
}

type ProtocolHandler interface {
	Broadcast(pi ProtocolInfo, b []byte, bt BroadcastType) error
	Multicast(pi ProtocolInfo, b []byte, role Role) error
	Unicast(pi ProtocolInfo, b []byte, id PeerID) error
}

type BroadcastType byte
type Role string

const (
	BROADCAST_ALL       BroadcastType = 0
	BROADCAST_NEIGHBOR  BroadcastType = 1
	ROLE_VALIDATOR      Role          = "validator"
	ROLE_SEED           Role          = "seed"
	ROLE_NORMAL         Role          = "normal"
)

type PeerID interface {
	Bytes() []byte
	Equal(PeerID) bool
	String() string
}

//TODO remove interface and using uint16
type ProtocolInfo interface {
	ID() byte
	Version() byte
	String() string
	Uint16() uint16
}

type NetworkTransport interface {
	Listen() error
	Close() error
	Dial(address string, channel string) error
	PeerID() PeerID
	Address() string
	SetListenAddress(address string) error
	GetListenAddress() string
}

//TODO remove interface and implement network.IsTemporaryError(error) bool
type NetworkError interface {
	error
	Temporary() bool // Is the error temporary?
}
