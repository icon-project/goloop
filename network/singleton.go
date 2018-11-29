package network

import (
	"errors"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var (
	ErrAlreadyListened           = errors.New("Already listened")
	ErrAlreadyClosed             = errors.New("Already closed")
	ErrAlreadyRegisteredReactor  = errors.New("Already registered reactor")
	ErrAlreadyRegisteredProtocol = errors.New("Already registered protocol")
	ErrNotRegisteredRole         = errors.New("Not registered role")
)
var (
	singletonTransport module.NetworkTransport
	singletonManagers  = make(map[string]module.NetworkManager)
	singletonConfig    *Config
)

var (
	singletonLoggerExcludes = []string{"Authenticator", "ChannelNegotiator", "PeerToPeer"}
)

const (
	DefaultTransportNet        = "tcp4"
	DefaultMembershipName      = ""
	DefaultPacketBufferSize    = 4096 //bufio.defaultBufSize=4096
	DefaultPacketPoolNumBucket = 10
	DefaultPacketPoolBucketLen = 10
	DefaultDiscoveryPeriodSec  = 1
	DefaultSeedPeriodSec       = 5
	DefaultSendDelay           = 1 * time.Second
	DefaultSendTaskTimeout     = 100 * time.Millisecond
)

var (
	PROTO_CONTOL     module.ProtocolInfo = protocolInfo(0x0000)
	PROTO_DEF_MEMBER module.ProtocolInfo = protocolInfo(0x0100)
)

var (
	PROTO_AUTH_KEY_REQ     module.ProtocolInfo = protocolInfo(0x0100)
	PROTO_AUTH_KEY_RESP    module.ProtocolInfo = protocolInfo(0x0200)
	PROTO_AUTH_SIGN_REQ    module.ProtocolInfo = protocolInfo(0x0300)
	PROTO_AUTH_SIGN_RESP   module.ProtocolInfo = protocolInfo(0x0400)
	PROTO_CHAN_JOIN_REQ    module.ProtocolInfo = protocolInfo(0x0500)
	PROTO_CHAN_JOIN_RESP   module.ProtocolInfo = protocolInfo(0x0600)
	PROTO_P2P_QUERY        module.ProtocolInfo = protocolInfo(0x0700)
	PROTO_P2P_QUERY_RESULT module.ProtocolInfo = protocolInfo(0x0800)
	PROTO_P2P_CONN_REQ     module.ProtocolInfo = protocolInfo(0x0900)
	PROTO_P2P_CONN_RESP    module.ProtocolInfo = protocolInfo(0x0A00)
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
		w, _ := common.WalletFromPrivateKey(c.PrivateKey)
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
