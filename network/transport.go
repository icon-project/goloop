package network

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
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
		l.Panicf("invalid P2P Address err:%+v", err)
	}
	transportLogger := l.WithFields(log.Fields{log.FieldKeyModule: "TP"})
	id := NewPeerIDFromAddress(w.Address())
	a := newAuthenticator(w, transportLogger)
	cn := newChannelNegotiator(na, id, transportLogger)
	pd := newPeerDispatcher(id, transportLogger, a, cn)
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
		if aead == SecureAeadSuiteNone {
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
