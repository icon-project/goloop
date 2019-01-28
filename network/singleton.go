package network

import (
	"crypto/elliptic"
	"errors"
	"math"
	"os"
	"time"
)

var (
	ErrAlreadyListened           = errors.New("already listened")
	ErrAlreadyClosed             = errors.New("already closed")
	ErrAlreadyRegisteredReactor  = errors.New("already registered reactor")
	ErrAlreadyRegisteredProtocol = errors.New("already registered protocol")
	ErrNotRegisteredProtocol     = errors.New("not registered protocol")
	ErrNotRegisteredRole         = errors.New("not registered role")
	ErrNotAvailable              = errors.New("not available")
	ErrQueueOverflow             = errors.New("queue overflow")
	ErrDuplicatedPacket          = errors.New("duplicated Packet")
)

var (
	ExcludeLoggers = []string{
		"Listener",
		"Dialer",
		"PeerDispatcher",
		"Authenticator",
		"ChannelNegotiator",
		//"PeerToPeer",
		"ProtocolHandler",
		"NetworkManager",
	}
)

const (
	DefaultTransportNet         = "tcp4"
	DefaultDialTimeout          = 5 * time.Second
	DefaultReceiveQueueSize     = 1000
	DefaultPacketBufferSize     = 4096 //bufio.defaultBufSize=4096
	DefaultPacketPayloadMax     = math.MaxInt32
	DefaultPacketPoolNumBucket  = 20
	DefaultPacketPoolBucketLen  = 500
	DefaultDiscoveryPeriod      = 2 * time.Second
	DefaultSeedPeriod           = 3 * time.Second
	DefaultMinSeed              = 1
	DefaultAlternateSendPeriod  = 1 * time.Second
	DefaultSendTimeout          = 5 * time.Second
	DefaultSendQueueMaxPriority = 7
	DefaultSendQueueSize        = 1000
	DefaultEventQueueSize       = 100
	DefaultPeerSendQueueSize    = 1000
	DefaultPeerPoolExpireSecond = 5
	DefaultUncleLimit           = 2
	DefaultChildrenLimit        = 1
	DefaultNephewLimit          = 2
	DefaultPacketRewriteLimit   = 10
	DefaultPacketRewriteDelay   = 100 * time.Millisecond
	DefaultRttAccuracy          = 10 * time.Millisecond
)

var (
	PROTO_CONTOL = protocolInfo(0x0000)
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
	PROTO_P2P_RTT_REQ      = protocolInfo(0x0B00)
	PROTO_P2P_RTT_RESP     = protocolInfo(0x0C00)
)

var (
	DefaultSecureEllipticCurve = elliptic.P256()
	DefaultSecureSuites        = []SecureSuite{
		SecureSuiteNone,
		SecureSuiteTls,
		SecureSuiteEcdhe,
	}
	DefaultSecureAeadSuites = []SecureAeadSuite{
		SecureAeadSuiteChaCha20Poly1305,
		SecureAeadSuiteAes128Gcm,
		SecureAeadSuiteAes256Gcm,
	}
	DefaultSecureKeyLogWriter = os.Stdout
)
