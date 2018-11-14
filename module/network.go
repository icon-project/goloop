package module

import (
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
)

type NetworkManager interface {
	//CreateIfNotExists and return Membership
	//if name == nil then return DefaultMembership with fixed PROTO_ID
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
	Multicast(subProtocol ProtocolInfo, bytes []byte, role Role) error
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

const (
	PeerIdSize = common.AddressBytes
)

//TODO refer to github.com/icon-project/goloop/common.NewAccountAddressFromPublicKey(pubKey)
type PeerId struct {
	*common.Address
}

func NewPeerId(b []byte) PeerId {
	return PeerId{common.NewAccountAddress(b)}
}

func NewPeerIdFromPublicKey(k *crypto.PublicKey) PeerId {
	return PeerId{common.NewAccountAddressFromPublicKey(k)}
}

func (pi PeerId) Copy(b []byte) {
	copy(b[:PeerIdSize], pi.ID())
}
func (pi PeerId) Equal(o PeerId) bool {
	return pi.Address.Equal(o.Address)
}
func (pi PeerId) IsNil() bool {
	return pi.Address == nil
}
func (pi PeerId) String() string {
	if pi.IsNil() {
		return ""
	} else {
		return pi.Address.String()
	}
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
