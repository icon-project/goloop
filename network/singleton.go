package network

import (
	"crypto/elliptic"
	"io"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	AlreadyListenedError = errors.CodeNetwork + iota
	AlreadyClosedError
	AlreadyDialingError
	AlreadyRegisteredReactorError
	AlreadyRegisteredProtocolError
	NotRegisteredReactorError
	NotRegisteredProtocolError
	NotRegisteredRoleError
	NotAuthorizedError
	NotAvailableError
	NotStartedError
	QueueOverflowError
	DuplicatedPacketError
	DuplicatedPeerError
)

var (
	ErrAlreadyListened           = errors.NewBase(AlreadyListenedError, "AlreadyListened")
	ErrAlreadyClosed             = errors.NewBase(AlreadyClosedError, "AlreadyClosed")
	ErrAlreadyDialing            = errors.NewBase(AlreadyDialingError, "AlreadyDialing")
	ErrAlreadyRegisteredReactor  = errors.NewBase(AlreadyRegisteredReactorError, "AlreadyRegisteredReactor")
	ErrAlreadyRegisteredProtocol = errors.NewBase(AlreadyRegisteredProtocolError, "AlreadyRegisteredProtocol")
	ErrNotRegisteredReactor      = errors.NewBase(NotRegisteredReactorError, "NotRegisteredReactor")
	ErrNotRegisteredProtocol     = errors.NewBase(NotRegisteredProtocolError, "NotRegisteredProtocol")
	ErrNotRegisteredRole         = errors.NewBase(NotRegisteredRoleError, "NotRegisteredRole")
	ErrNotAuthorized             = errors.NewBase(NotAuthorizedError, "NotAuthorized")
	ErrNotAvailable              = errors.NewBase(NotAvailableError, "NotAvailable")
	ErrNotStarted                = errors.NewBase(NotStartedError, "NotStarted")
	ErrQueueOverflow             = errors.NewBase(QueueOverflowError, "QueueOverflow")
	ErrDuplicatedPacket          = errors.NewBase(DuplicatedPacketError, "DuplicatedPacket")
	ErrDuplicatedPeer            = errors.NewBase(DuplicatedPeerError, "DuplicatedPeer")
	ErrIllegalArgument           = errors.ErrIllegalArgument
)

const (
	LoggerFieldKeySubModule = "sub"
)

const (
	DefaultTransportNet         = "tcp4"
	DefaultDialTimeout          = 5 * time.Second
	DefaultReceiveQueueSize     = 1000
	DefaultPacketBufferSize     = 4096 //bufio.defaultBufSize=4096
	DefaultPacketPayloadMax     = 1024 * 1024
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
	DefaultFailureQueueSize     = 100
	DefaultPeerSendQueueSize    = 1000
	DefaultPeerPoolExpireSecond = 5
	DefaultUncleLimit           = 1
	DefaultChildrenLimit        = 10
	DefaultNephewLimit          = 10
	DefaultPacketRewriteLimit   = 10
	DefaultPacketRewriteDelay   = 100 * time.Millisecond
	DefaultRttAccuracy          = 10 * time.Millisecond
	DefaultFailureNodeMin       = 2
	DefaultSelectiveFloodingAdd = 1
	DefaultSimplePeerIDSize     = 4
	UsingSelectiveFlooding      = true
	DefaultDuplicatedPeerTime   = 1 * time.Second
	DefaultMaxRetryClose		= 10
)

var (
	PROTO_CONTOL = module.ProtocolInfo(0x0000)
)

var (
	PROTO_AUTH_KEY_REQ     = module.ProtocolInfo(0x0100)
	PROTO_AUTH_KEY_RESP    = module.ProtocolInfo(0x0200)
	PROTO_AUTH_SIGN_REQ    = module.ProtocolInfo(0x0300)
	PROTO_AUTH_SIGN_RESP   = module.ProtocolInfo(0x0400)
	PROTO_CHAN_JOIN_REQ    = module.ProtocolInfo(0x0500)
	PROTO_CHAN_JOIN_RESP   = module.ProtocolInfo(0x0600)
	PROTO_P2P_QUERY        = module.ProtocolInfo(0x0700)
	PROTO_P2P_QUERY_RESULT = module.ProtocolInfo(0x0800)
	PROTO_P2P_CONN_REQ     = module.ProtocolInfo(0x0900)
	PROTO_P2P_CONN_RESP    = module.ProtocolInfo(0x0A00)
	PROTO_P2P_RTT_REQ      = module.ProtocolInfo(0x0B00)
	PROTO_P2P_RTT_RESP     = module.ProtocolInfo(0x0C00)
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
	DefaultSecureKeyLogWriter io.Writer
)
