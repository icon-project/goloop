package network

import (
	"container/list"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

type transport struct {
	l       *Listener
	address NetAddress
	a       *Authenticator
	cn      *ChannelNegotiator
	pd      *PeerDispatcher
	dMap    map[string]*Dialer
	logger  log.Logger
}

func NewTransport(address string, w module.Wallet, l log.Logger) module.NetworkTransport {
	na := NetAddress(address)
	if err := na.Validate(); err != nil {
		l.Panicf("invalid P2P Address err:%+v",err)
	}
	transportLogger := l.WithFields(log.Fields{log.FieldKeyModule: "TP"})
	a := newAuthenticator(w, transportLogger)
	cn := newChannelNegotiator(na, transportLogger)
	pd := newPeerDispatcher(NewPeerIDFromAddress(w.Address()), transportLogger, cn, a)
	listener := newListener(address, pd.onAccept, transportLogger)
	t := &transport{
		l:       listener,
		address: na,
		a:       a,
		cn:      cn,
		pd:      pd,
		dMap:    make(map[string]*Dialer),
		logger:  transportLogger,
	}
	return t
}

func (t *transport) Listen() error {
	return t.l.Listen()
}

func (t *transport) Close() error {
	return t.l.Close()
}
func (t *transport) Dial(address string, channel string) error {
	d := t.GetDialer(channel)
	return d.Dial(address)
}

func (t *transport) PeerID() module.PeerID {
	return t.pd.self
}

func (t *transport) Address() string {
	return string(t.address)
}

func (t *transport) SetListenAddress(address string) error {
	return t.l.SetAddress(address)
}

func (t *transport) GetListenAddress() string {
	return t.l.Address()
}

func (t *transport) GetDialer(channel string) *Dialer {
	d, ok := t.dMap[channel]
	if !ok {
		d = newDialer(channel, t.pd.onConnect)
		t.dMap[channel] = d
	}
	return d
}

func (t *transport) SetSecureSuites(channel string, secureSuites string) error {
	if secureSuites == "" {
		return t.a.SetSecureSuites(channel, nil)
	}
	ss := strings.Split(secureSuites, ",")
	suites := make([]SecureSuite, len(ss))
	for i, s := range ss {
		suite := SecureSuiteFromString(s)
		if suite == SecureSuiteUnknown {
			return fmt.Errorf("parse SecureSuite error from %s", s)
		}
		suites[i] = suite
	}
	return t.a.SetSecureSuites(channel, suites)
}

func (t *transport) GetSecureSuites(channel string) string {
	suites := t.a.GetSecureSuites(channel)

	s := make([]string, len(suites))
	for i, suite := range suites {
		s[i] = suite.String()
	}
	return strings.Join(s, ",")
}

func (t *transport) SetSecureAeads(channel string, secureAeads string) error {
	if secureAeads == "" {
		return t.a.SetSecureAeads(channel, nil)
	}
	ss := strings.Split(secureAeads, ",")
	aeads := make([]SecureAeadSuite, len(ss))
	for i, s := range ss {
		aead := SecureAeadSuiteFromString(s)
		if aead == SecureAeadSuiteUnknown {
			return fmt.Errorf("parse SecureAeadSuite error from %s", s)
		}
		aeads[i] = aead
	}
	return t.a.SetSecureAeads(channel, aeads)
}

func (t *transport) GetSecureAeads(channel string) string {
	aeads := t.a.GetSecureAeads(channel)

	s := make([]string, len(aeads))
	for i, aead := range aeads {
		s[i] = aead.String()
	}
	return strings.Join(s, ",")
}

type Listener struct {
	address  string
	ln       net.Listener
	mtx      sync.Mutex
	closeCh  chan bool
	onAccept acceptCbFunc
	//log
	logger log.Logger
}

type acceptCbFunc func(conn net.Conn)

func newListener(address string, cbFunc acceptCbFunc, l log.Logger) *Listener {
	return &Listener{
		address:  address,
		onAccept: cbFunc,
		logger:   l.WithFields(log.Fields{LoggerFieldKeySubModule: "listener"}),
	}
}

func (l *Listener) Address() string {
	if l.ln == nil {
		return l.address
	}
	la := l.ln.Addr()
	return la.String()
}

func (l *Listener) SetAddress(address string) error {
	defer l.mtx.Unlock()
	l.mtx.Lock()

	if l.ln != nil {
		return ErrAlreadyListened
	}

	l.address = address
	return nil
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

	if l.ln == nil {
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
			l.logger.Infoln("acceptRoutine", err)
			return
		}
		l.onAccept(conn)
	}
}

type Dialer struct {
	onConnect connectCbFunc
	channel   string
	dialing   *Set
}

type connectCbFunc func(conn net.Conn, addr string, d *Dialer)

func newDialer(channel string, cbFunc connectCbFunc) *Dialer {
	return &Dialer{
		onConnect: cbFunc,
		channel:   channel,
		dialing:   NewSet(),
	}
}

func (d *Dialer) Dial(addr string) error {
	if !d.dialing.Add(addr) {
		return ErrAlreadyDialing
	}
	conn, err := net.DialTimeout(DefaultTransportNet, addr, DefaultDialTimeout)
	_ = d.dialing.Remove(addr)
	if err != nil {
		return err
	}
	d.onConnect(conn, addr, d)
	return nil
}

type PeerHandler interface {
	onPeer(p *Peer)
	onPacket(pkt *Packet, p *Peer)
	onError(err error, p *Peer, pkt *Packet)
	onClose(p *Peer)
	setNext(ph PeerHandler)
	setSelfPeerID(id module.PeerID)
}

type peerHandler struct {
	next PeerHandler
	self module.PeerID
	//log
	logger log.Logger
}

func newPeerHandler(l log.Logger) *peerHandler {
	return &peerHandler{logger: l}
}

func (ph *peerHandler) onPeer(p *Peer) {
	ph.logger.Traceln("onPeer", p)
	ph.nextOnPeer(p)
}

func (ph *peerHandler) nextOnPeer(p *Peer) {
	if ph.next != nil {
		p.setPacketCbFunc(ph.next.onPacket)
		p.setErrorCbFunc(ph.next.onError)
		p.setCloseCbFunc(ph.next.onClose)
		ph.next.onPeer(p)
	}
}

func (ph *peerHandler) onError(err error, p *Peer, pkt *Packet) {
	ph.logger.Traceln("onError", err, p)
	p.CloseByError(err)
}

func (ph *peerHandler) onClose(p *Peer) {
	ph.logger.Traceln("onClose", p)
}

func (ph *peerHandler) setNext(next PeerHandler) {
	ph.next = next
}

func (ph *peerHandler) setSelfPeerID(id module.PeerID) {
	ph.self = id
}

func (ph *peerHandler) sendMessage(pi module.ProtocolInfo, m interface{}, p *Peer) {
	pkt := newPacket(pi, ph.encode(m), ph.self)
	err := p.sendDirect(pkt)
	if err != nil {
		ph.logger.Infoln("sendMessage", err)
		p.CloseByError(err)
	} else {
		ph.logger.Traceln("sendMessage", m, p)
	}
}

func (ph *peerHandler) encode(v interface{}) []byte {
	b := make([]byte, DefaultPacketBufferSize)
	enc := codec.MP.NewEncoderBytes(&b)
	if err := enc.Encode(v); err != nil {
		log.Panicf("fail to encode object v=%+v err=%+v", v, err)
	}
	return b
}

func (ph *peerHandler) decode(b []byte, v interface{}) {
	codec.MP.MustUnmarshalFromBytes(b, v)
}

type PeerDispatcher struct {
	*peerHandler
	peerHandlers *list.List
	p2pMap       map[string]*PeerToPeer
	mtx          sync.RWMutex

	mtr *metric.NetworkMetric
}

func newPeerDispatcher(id module.PeerID, l log.Logger, peerHandlers ...PeerHandler) *PeerDispatcher {
	pd := &PeerDispatcher{
		peerHandlers: list.New(),
		p2pMap:       make(map[string]*PeerToPeer),
		peerHandler:  newPeerHandler(l),
		mtr:          metric.NewNetworkMetric(metric.DefaultMetricContext()),
	}
	// pd.peerHandler.codecHandle.MapType = reflect.TypeOf(map[string]interface{}(nil))
	pd.setSelfPeerID(id)

	pd.registerPeerHandler(pd, true)
	for _, ph := range peerHandlers {
		pd.registerPeerHandler(ph, true)
	}
	return pd
}

func (pd *PeerDispatcher) registerPeerToPeer(p2p *PeerToPeer) bool {
	defer pd.mtx.Unlock()
	pd.mtx.Lock()

	if _, ok := pd.p2pMap[p2p.channel]; ok {
		return false
	}
	pd.p2pMap[p2p.channel] = p2p
	return true
}

func (pd *PeerDispatcher) unregisterPeerToPeer(p2p *PeerToPeer) bool {
	defer pd.mtx.Unlock()
	pd.mtx.Lock()
	if t, ok := pd.p2pMap[p2p.channel]; !ok || t != p2p {
		return false
	}
	delete(pd.p2pMap, p2p.channel)
	return true
}

func (pd *PeerDispatcher) getPeerToPeer(channel string) *PeerToPeer {
	defer pd.mtx.RUnlock()
	pd.mtx.RLock()

	return pd.p2pMap[channel]
}

func (pd *PeerDispatcher) registerPeerHandler(ph PeerHandler, pushBack bool) {
	pd.logger.Traceln("registerPeerHandler", ph, pushBack)
	if pushBack {
		elm := pd.peerHandlers.PushBack(ph)
		if prev := elm.Prev(); prev != nil {
			ph.setNext(prev.Value.(PeerHandler))
			ph.setSelfPeerID(pd.self)
		}
	} else {
		f := pd.peerHandlers.Front()
		elm := pd.peerHandlers.InsertAfter(ph, f)
		pd.setNext(ph)
		ph.setSelfPeerID(pd.self)
		if next := elm.Next(); next != nil {
			next.Value.(PeerHandler).setNext(ph)
		}
	}
}

//callback from Listener.acceptRoutine
func (pd *PeerDispatcher) onAccept(conn net.Conn) {
	pd.logger.Traceln("onAccept", conn.LocalAddr(), "<-", conn.RemoteAddr())
	p := newPeer(conn, nil, true, pd.logger)
	pd.dispatchPeer(p)
}

//callback from Dialer.Connect
func (pd *PeerDispatcher) onConnect(conn net.Conn, addr string, d *Dialer) {
	pd.logger.Traceln("onConnect", conn.LocalAddr(), "->", conn.RemoteAddr())
	p := newPeer(conn, nil, false, pd.logger)
	p.channel = d.channel
	p.netAddress = NetAddress(addr)
	pd.dispatchPeer(p)
}

func (pd *PeerDispatcher) dispatchPeer(p *Peer) {
	elm := pd.peerHandlers.Back()
	ph := elm.Value.(PeerHandler)
	p.setMetric(pd.mtr)
	p.setPacketCbFunc(ph.onPacket)
	p.setErrorCbFunc(ph.onError)
	p.setCloseCbFunc(ph.onClose)
	ph.onPeer(p)
}

//callback from PeerHandler.nextOnPeer
func (pd *PeerDispatcher) onPeer(p *Peer) {
	pd.logger.Traceln("onPeer", p)
	if p2p := pd.getPeerToPeer(p.channel); p2p != nil {
		p.setMetric(p2p.mtr)
		p.setPacketCbFunc(p2p.onPacket)
		p.setErrorCbFunc(p2p.onError)
		p.setCloseCbFunc(p2p.onClose)
		p2p.onPeer(p)
	} else {
		err := fmt.Errorf("not exists PeerToPeer[%s]", p.channel)
		p.CloseByError(err)
	}
}

func (pd *PeerDispatcher) onError(err error, p *Peer, pkt *Packet) {
	pd.peerHandler.onError(err, p, pkt)
}

//callback from Peer.receiveRoutine
func (pd *PeerDispatcher) onPacket(pkt *Packet, p *Peer) {
	//TODO dispatcher.message_dump
	pd.logger.Traceln("onPacket", pkt)
}

func (pd *PeerDispatcher) onClose(p *Peer) {
	pd.peerHandler.onClose(p)
}
