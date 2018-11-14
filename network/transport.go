package network

import (
	"container/list"
	"errors"
	"log"
	"net"
	"sync"

	"github.com/icon-project/goloop/module"
)

const (
	TRANSPORT_NET = "tcp4"
)

var (
	ErrAlreadyListened = errors.New("Already listened")
	ErrAlreadyClosed   = errors.New("Already closed")
	l                  *Listener
	pd                 *PeerDispatcher
)

type Listener struct {
	address     string
	ln          net.Listener
	mtx         sync.Mutex
	closeCh     chan bool
	peerToPeers []*PeerToPeer
	onAccept    acceptCbFunc
}

type acceptCbFunc func(conn net.Conn)

func GetListener() *Listener {
	if l == nil {
		c := GetConfig()
		l = newListener(c.ListenAddress)
	}
	return l
}

func newListener(address string) *Listener {
	return &Listener{
		address:  address,
		onAccept: GetPeerDispatcher().onAccept,
	}
}

func (l *Listener) Listen() error {
	defer l.mtx.Unlock()
	l.mtx.Lock()
	if l.ln != nil {
		return ErrAlreadyListened
	}
	ln, err := net.Listen(TRANSPORT_NET, l.address)
	if err != nil {
		return err
	}
	l.ln = ln
	l.closeCh = make(chan bool)
	go l.acceptRoutine()
	return nil
}

func (l *Listener) Close() error {
	defer l.mtx.Unlock()
	l.mtx.Lock()

	if l.ln != nil {
		return ErrAlreadyClosed
	}
	err := l.ln.Close()
	if err != nil {
		return err
	}
	<-l.closeCh

	l.ln = nil
	return nil
}

func (l *Listener) acceptRoutine() {
	defer close(l.closeCh)

	for {
		conn, err := l.ln.Accept()
		if err != nil {
			log.Println("acceptRoutine", err)
			return
		}
		l.onAccept(conn)
	}
}

type Dialer struct {
	onConnect connectCbFunc
	channel   string
	conn      net.Conn
}

type connectCbFunc func(conn net.Conn, d *Dialer)

func NewDialer(channel string) *Dialer {
	return &Dialer{
		onConnect: GetPeerDispatcher().onConnect,
		channel:   channel,
	}
}

func (d *Dialer) Dial(address string) error {
	conn, err := net.Dial(TRANSPORT_NET, address)
	if err != nil {
		return err
	}
	d.conn = conn
	d.onConnect(conn, d)
	return nil
}

type PeerHandler interface {
	onPeer(p *Peer)
	onPacket(pkt *Packet, p *Peer)
	setNext(ph PeerHandler)
	setSelfPeerId(peerId module.PeerId)
}

type peerHandler struct {
	next PeerHandler
	self module.PeerId
}

func (ph *peerHandler) onPeer(p *Peer) {
	ph.nextOnPeer(p)
}

func (ph *peerHandler) nextOnPeer(p *Peer) {
	if ph.next != nil {
		p.setPacketCbFunc(ph.next.onPacket)
		ph.next.onPeer(p)
	}
}

func (ph *peerHandler) setNext(next PeerHandler) {
	ph.next = next
}

func (ph *peerHandler) setSelfPeerId(peerId module.PeerId) {
	ph.self = peerId
}

func (ph *peerHandler) sendPacket(pkt *Packet, p *Peer) error {
	log.Println("peerHandler.sendPacket", pkt)
	if pkt.src.IsNil() {
		pkt.src = ph.self
	}
	return p.sendPacket(pkt)
}

type PeerDispatcher struct {
	peerHandler
	peerHandlers *list.List
	peerToPeers  map[string]*PeerToPeer
}

func GetPeerDispatcher() *PeerDispatcher {
	if pd == nil {
		c := GetConfig()
		pd = newPeerDispatcher(
			module.NewPeerIdFromPublicKey(c.PublicKey),
			GetChannelNegotiator(),
			GetAuthenticator())
	}
	return pd
}

func newPeerDispatcher(selfPeerId module.PeerId, peerHandlers ...PeerHandler) *PeerDispatcher {
	pd := &PeerDispatcher{
		peerHandlers: list.New(),
		peerToPeers:  make(map[string]*PeerToPeer),
	}
	pd.self = selfPeerId
	log.Println("PeerDispatcher.self", selfPeerId)
	pd.registPeerHandler(pd)

	for _, ph := range peerHandlers {
		pd.registPeerHandler(ph)
	}
	return pd
}

func (pd *PeerDispatcher) registPeerToPeer(p2p *PeerToPeer) {
	pd.peerToPeers[p2p.channel] = p2p
}

func (pd *PeerDispatcher) registPeerHandler(ph PeerHandler) {
	elm := pd.peerHandlers.PushBack(ph)
	if prev := elm.Prev(); prev != nil {
		ph.setNext(prev.Value.(PeerHandler))
		ph.setSelfPeerId(pd.self)
	}
}

//callback from Listener.acceptRoutine
func (pd *PeerDispatcher) onAccept(conn net.Conn) {
	log.Println("PeerDispatcher.onAccept", conn.RemoteAddr())
	p := newPeer(conn, nil, true)
	pd.dispatchPeer(p)
}

//callback from Dialer.Connect
func (pd *PeerDispatcher) onConnect(conn net.Conn, d *Dialer) {
	log.Println("PeerDispatcher.onConnect", conn.RemoteAddr())
	p := newPeer(conn, nil, false)
	p.channel = d.channel
	pd.dispatchPeer(p)
}

func (pd *PeerDispatcher) dispatchPeer(p *Peer) {
	elm := pd.peerHandlers.Back()
	ph := elm.Value.(PeerHandler)
	p.setPacketCbFunc(ph.onPacket)
	ph.onPeer(p)
}

//call PeerHandler.nextOnPeer, peerHandlers.
func (pd *PeerDispatcher) onPeer(p *Peer) {
	log.Println("PeerDispatcher.onPeer", p)
	if p2p, ok := pd.peerToPeers[p.channel]; ok {
		p.setPacketCbFunc(p2p.onPacket)
		p2p.onPeer(p)
	} else {
		log.Println("Not exists PeerToPeer[", p.channel, "], try close")
		p.conn.Close()
	}
}

func (pd *PeerDispatcher) onPacket(pkt *Packet, p *Peer) {
	log.Println("PeerDispatcher.onPacket", pkt)
}
