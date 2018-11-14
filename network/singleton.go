package network

import (
	"errors"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

var (
	ErrAlreadyListened = errors.New("Already listened")
	ErrAlreadyClosed   = errors.New("Already closed")
)
var (
	transportListener        *Listener
	transportPeerDispatcher  *PeerDispatcher
	transportDialers         = make(map[string]*Dialer)
	networkManagers          = make(map[string]module.NetworkManager)
	handlerChannelNegotiator *ChannelNegotiator
	handlerAuthenticator     *Authenticator
	singletonConfig          *Config
)

const (
	DefaultTransportNet     = "tcp4"
	DefaultMembershipName   = ""
	DefaultPacketBufferSize = 4096 //bufio.defaultBufSize=4096
)

const (
	PROTO_CONTOL     = 0x0000
	PROTO_DEF_MEMBER = 0x0100
)

const (
	PROTO_AUTH_HS1 = 0x0101
	PROTO_AUTH_HS2 = 0x0201
	PROTO_AUTH_HS3 = 0x0301
	PROTO_AUTH_HS4 = 0x0401
)
const (
	PROTO_CHAN_JOIN_REQ  = 0x0501
	PROTO_CHAN_JOIN_RESP = 0x0601
)

const (
	PROTO_P2P_QUERY        = 0x0701
	PROTO_P2P_QUERY_RESULT = 0x0801
)

type Config struct {
	ListenAddress string
	PrivateKey    *crypto.PrivateKey
	PublicKey     *crypto.PublicKey
}

func GetConfig() *Config {
	if singletonConfig == nil {
		//TODO Read from file or DB
		priK, pubK := crypto.GenerateKeyPair()
		singletonConfig = &Config{
			ListenAddress: "127.0.0.1:8080",
			PrivateKey:    priK,
			PublicKey:     pubK,
		}

	}
	return singletonConfig
}

func GetListener() *Listener {
	if transportListener == nil {
		c := GetConfig()
		transportListener = newListener(c.ListenAddress, GetPeerDispatcher().onAccept)
	}
	return transportListener
}

func GetDialer(channel string) *Dialer {
	d, ok := transportDialers[channel]
	if !ok {
		d = newDialer(channel, GetPeerDispatcher().onConnect)
		transportDialers[channel] = d
	}
	return d
}

func GetPeerDispatcher() *PeerDispatcher {
	if transportPeerDispatcher == nil {
		c := GetConfig()
		transportPeerDispatcher = newPeerDispatcher(
			NewPeerIdFromPublicKey(c.PublicKey),
			GetChannelNegotiator(),
			GetAuthenticator())
	}
	return transportPeerDispatcher
}

func GetNetworkManager(channel string) module.NetworkManager {
	nm, ok := networkManagers[channel]
	if !ok {
		l := GetListener()
		pd := GetPeerDispatcher()
		m := newManager(channel, pd.self, NetAddress(l.address))
		pd.registPeerToPeer(m.peerToPeer)
		nm = m
		networkManagers[channel] = nm
	}
	return nm
}

func GetChannelNegotiator() *ChannelNegotiator {
	if handlerChannelNegotiator == nil {
		handlerChannelNegotiator = newChannelNegotiator()
	}
	return handlerChannelNegotiator
}

func GetAuthenticator() *Authenticator {
	if handlerAuthenticator == nil {
		c := GetConfig()
		handlerAuthenticator = newAuthenticator(c.PrivateKey, c.PublicKey)
	}
	return handlerAuthenticator
}
