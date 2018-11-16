package network

import (
	"container/list"
	"log"
	"net"
	"sync"

	"github.com/icon-project/goloop/module"
)

type Listener struct {
	address  string
	ln       net.Listener
	mtx      sync.Mutex
	closeCh  chan bool
	onAccept acceptCbFunc
}

type acceptCbFunc func(conn net.Conn)

func newListener(address string, cbFunc acceptCbFunc) *Listener {
	return &Listener{
		address:  address,
		onAccept: cbFunc,
	}
}

func (l *Listener) Listen() error {
	defer l.mtx.Unlock()
	l.mtx.Lock()
	if l.ln != nil {
		return ErrAlreadyListened
	}
	ln, err := net.Listen(DefaultTransportNet, l.address)
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

type connectCbFunc func(conn net.Conn, addr string, d *Dialer)

func newDialer(channel string, cbFunc connectCbFunc) *Dialer {
	return &Dialer{
		onConnect: cbFunc,
		channel:   channel,
	}
}

func (d *Dialer) Dial(addr string) error {
	conn, err := net.Dial(DefaultTransportNet, addr)
	if err != nil {
		return err
	}
	d.conn = conn
	d.onConnect(conn, addr, d)
	return nil
}

type PeerHandler interface {
	onPeer(p *Peer)
	onPacket(pkt *Packet, p *Peer)
	onError(err error, p *Peer)
	setNext(ph PeerHandler)
	setSelfPeerID(id module.PeerID)
}

type peerHandler struct {
	next PeerHandler
	self module.PeerID
}

func (ph *peerHandler) onPeer(p *Peer) {
	ph.nextOnPeer(p)
}

func (ph *peerHandler) nextOnPeer(p *Peer) {
	if ph.next != nil {
		p.setPacketCbFunc(ph.next.onPacket)
		p.setErrorCbFunc(ph.next.onError)
		ph.next.onPeer(p)
	}
}

func (ph *peerHandler) onError(err error, p *Peer) {
	log.Println("peerHandler.onError", err, p)
	err = p.conn.Close()
	if err != nil {
		log.Println("peerHandler.onError p.conn.Close()", err)
	}
}

func (ph *peerHandler) setNext(next PeerHandler) {
	ph.next = next
}

func (ph *peerHandler) setSelfPeerID(id module.PeerID) {
	ph.self = id
}

func (ph *peerHandler) sendPacket(pkt *Packet, p *Peer) error {
	if pkt.src == nil {
		pkt.src = ph.self
	}
	//log.Println("peerHandler.sendPacket", pkt)
	return p.sendPacket(pkt)
}

type PeerDispatcher struct {
	peerHandler
	peerHandlers *list.List
	peerToPeers  map[string]*PeerToPeer
}

func newPeerDispatcher(selfPeerId module.PeerID, peerHandlers ...PeerHandler) *PeerDispatcher {
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
		ph.setSelfPeerID(pd.self)
	}
}

//callback from Listener.acceptRoutine
func (pd *PeerDispatcher) onAccept(conn net.Conn) {
	log.Println("PeerDispatcher.onAccept", conn.RemoteAddr())
	p := newPeer(conn, nil, true)
	pd.dispatchPeer(p)
}

//callback from Dialer.Connect
func (pd *PeerDispatcher) onConnect(conn net.Conn, addr string, d *Dialer) {
	log.Println("PeerDispatcher.onConnect", conn.RemoteAddr())
	p := newPeer(conn, nil, false)
	p.channel = d.channel
	p.netAddress = NetAddress(addr)
	pd.dispatchPeer(p)
}

func (pd *PeerDispatcher) dispatchPeer(p *Peer) {
	elm := pd.peerHandlers.Back()
	ph := elm.Value.(PeerHandler)
	p.setPacketCbFunc(ph.onPacket)
	p.setErrorCbFunc(ph.onError)
	ph.onPeer(p)
}

//callback from PeerHandler.nextOnPeer
func (pd *PeerDispatcher) onPeer(p *Peer) {
	log.Println("PeerDispatcher.onPeer", p)
	if p2p, ok := pd.peerToPeers[p.channel]; ok {
		p.setPacketCbFunc(p2p.onPacket)
		p.setErrorCbFunc(p2p.onError)
		p2p.onPeer(p)
	} else {
		log.Println("Not exists PeerToPeer[", p.channel, "], try close")
		err := p.conn.Close()
		if err != nil {
			log.Println("PeerDispatcher.onPeer p.conn.Close()", err)
		}
	}
}

//TODO callback from Peer.sendRoutine or Peer.receiveRoutine
func (pd *PeerDispatcher) onError(err error, p *Peer) {
	log.Println("PeerDispatcher.onError", err, p)
	err = p.conn.Close()
	if err != nil {
		log.Println("PeerDispatcher.onError p.conn.Close()", err)
	}
}

//callback from Peer.receiveRoutine
func (pd *PeerDispatcher) onPacket(pkt *Packet, p *Peer) {
	log.Println("PeerDispatcher.onPacket", pkt)
}
