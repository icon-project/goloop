package network

import (
	"errors"
	"math"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var (
	ErrAlreadyListened           = errors.New("already listened")
	ErrAlreadyClosed             = errors.New("already closed")
	ErrAlreadyRegisteredReactor  = errors.New("already registered reactor")
	ErrAlreadyRegisteredProtocol = errors.New("already registered protocol")
	ErrNotRegisteredRole         = errors.New("not registered role")
	ErrNotAvailable              = errors.New("not available")
	ErrQueueOverflow             = errors.New("queue overflow")
	ErrDuplicatedPacket          = errors.New("duplicated Packet")
	ErrNilPacket                 = errors.New("nil Packet")
)
var (
	singletonTransport module.NetworkTransport
	singletonManagers  = make(map[string]module.NetworkManager)
	singletonConfig    *Config
)

var (
	singletonLoggerExcludes = []string{
		"Listener",
		"Dialer",
		"PeerDispatcher",
		"Authenticator",
		"ChannelNegotiator",
		"PeerToPeer",
		"Membership",
		"NetworkManager",
	}
)

const (
	DefaultTransportNet         = "tcp4"
	DefaultMembershipName       = ""
	DefaultReceiveQueueSize     = 1000
	DefaultPacketBufferSize     = 4096 //bufio.defaultBufSize=4096
	DefaultPacketPayloadMax     = math.MaxInt32
	DefaultPacketPoolNumBucket  = 20
	DefaultPacketPoolBucketLen  = 100
	DefaultDiscoveryPeriod      = 2 * time.Second
	DefaultSeedPeriod           = 3 * time.Second
	DefaultAlternateSendPeriod  = 1 * time.Second
	DefaultSendTimeout          = 1 * time.Second
	DefaultSendQueueSize        = 1000
	DefaultEventQueueSize       = 100
	DefaultPeerSendQueueSize    = 1000
	DefaultPeerPoolExpireSecond = 5
	DefaultUncleLimit           = 1
	DefaultPacketRewriteLimit   = 10
	DefaultPacketRewriteDelay   = 100 * time.Millisecond
)

var (
	PROTO_CONTOL     = protocolInfo(0x0000)
	PROTO_DEF_MEMBER = protocolInfo(0x0100)
)

var (
	PROTO_AUTH_KEY_REQ     = protocolInfo(0x0100)
	PROTO_AUTH_KEY_RESP    = protocolInfo(0x0200)
	PROTO_AUTH_SIGN_REQ    = protocolInfo(0x0300)
	PROTO_AUTH_SIGN_RESP   = protocolInfo(0x0400)
	PROTO_CHAN_JOIN_REQ    = protocolInfo(0x0500)
	PROTO_CHAN_JOIN_RESP   = protocolInfo(0x0600)
	PROTO_P2P_QUERY        = protocolInfo(0x0700)
	PROTO_P2P_QUERY_RESULT = protocolInfo(0x0800)
	PROTO_P2P_CONN_REQ     = protocolInfo(0x0900)
	PROTO_P2P_CONN_RESP    = protocolInfo(0x0A00)
)

type Config struct {
	ListenAddress string
	SeedAddress   string
	RoleSeed      bool
	RoleRoot      bool
	PrivateKey    *crypto.PrivateKey
}

func GetConfig() *Config {
	if singletonConfig == nil {
		//TODO Read from file or DB
		priK, _ := crypto.GenerateKeyPair()
		singletonConfig = &Config{
			ListenAddress: "127.0.0.1:8080",
			PrivateKey:    priK,
		}

	}
	return singletonConfig
}

func GetTransport() module.NetworkTransport {
	if singletonTransport == nil {
		c := GetConfig()
		w, _ := common.NewWalletFromPrivateKey(c.PrivateKey)
		singletonTransport = NewTransport(c.ListenAddress, w)
	}
	return singletonTransport
}

func GetManager(channel string) module.NetworkManager {
	nm, ok := singletonManagers[channel]
	if !ok {
		c := GetConfig()
		t := GetTransport()
		m := NewManager(channel, t)

		r := PeerRoleFlag(p2pRoleNone)
		if c.RoleSeed {
			r.SetFlag(p2pRoleSeed)
		}
		if c.RoleRoot {
			r.SetFlag(p2pRoleRoot)
		}
		m.(*manager).p2p.setRole(r)
		if c.SeedAddress != "" {
			m.(*manager).p2p.seeds.Add(NetAddress(c.SeedAddress))
		}
		nm = m
		singletonManagers[channel] = m
	}
	return nm
}
