package network

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

type Peer struct {
	id         module.PeerID
	netAddress NetAddress
	secureKey  *secureKey
	//
	conn        net.Conn
	reader      *PacketReader
	writer      *PacketWriter
	q           *PriorityQueue
	onPacket    packetCbFunc
	onError     errorCbFunc
	onClose     closeCbFunc
	timestamp   time.Time
	pool        *TimestampPool
	close       chan error
	closed      bool
	closeReason []string
	closeErr    []error
	mtx         sync.Mutex
	once        sync.Once
	//
	incomming bool
	channel   string
	rtt       PeerRTT
	connType  PeerConnectionType
	role      PeerRoleFlag
	roleMtx   sync.RWMutex
	children  []NetAddress
	nephews    int
	//
}

type packetCbFunc func(pkt *Packet, p *Peer)
type errorCbFunc func(err error, p *Peer, pkt *Packet)
type closeCbFunc func(p *Peer)

//TODO define netAddress as IP:Port
type NetAddress string

//TODO define PeerRTT,
type PeerRTT struct {
	last time.Duration
	avg  time.Duration
	st   time.Time
	et   time.Time
}

func NewPeerRTT() *PeerRTT {
	return &PeerRTT{}
}

func (r *PeerRTT) Start() time.Time {
	r.st = time.Now()
	return r.st
}

func (r *PeerRTT) Stop() time.Time {
	r.et = time.Now()
	r.last = r.et.Sub(r.st)

	//exponential weighted moving average model
	//avg = (1-0.125)*avg + 0.125*last
	if r.avg > 0 {
		fv := 0.875*float64(r.avg) + 0.125*float64(r.last)
		r.avg = time.Duration(fv)
	} else {
		r.avg = r.last
	}
	return r.et
}

func (r *PeerRTT) Last(d time.Duration) float64 {
	fv := float64(r.last) / float64(d)
	return fv
}

func (r *PeerRTT) Avg(d time.Duration) float64 {
	fv := float64(r.avg) / float64(d)
	return fv
}

func (r *PeerRTT) String() string {
	return fmt.Sprintf("{last:%v,avg:%v}", r.last.String(), r.avg.String())
}

const (
	p2pRoleNone     = 0x00
	p2pRoleSeed     = 0x01
	p2pRoleRoot     = 0x02
	p2pRoleRootSeed = 0x03
)

//PeerRoleFlag as BitFlag MSB[_,_,_,_,_,_,Root,Seed]LSB
//TODO remove p2pRoleRootSeed
type PeerRoleFlag byte

func (pr *PeerRoleFlag) Has(o PeerRoleFlag) bool {
	return (*pr)&o == o
}
func (pr *PeerRoleFlag) SetFlag(o PeerRoleFlag) {
	*pr |= o
}
func (pr *PeerRoleFlag) UnSetFlag(o PeerRoleFlag) {
	*pr &= ^o
}

const (
	p2pConnTypeNone = iota
	p2pConnTypeParent
	p2pConnTypeChildren
	p2pConnTypeUncle
	p2pConnTypeNephew
	p2pConnTypeFriend
)

var (
	strPeerConnectionType = []string{
		"Orphanage",
		"Parent",
		"Children",
		"Uncle",
		"Nephew",
		"Friend",
	}
)

type PeerConnectionType byte

func newPeer(conn net.Conn, cbFunc packetCbFunc, incomming bool) *Peer {
	p := &Peer{
		conn:        conn,
		reader:      NewPacketReader(conn),
		writer:      NewPacketWriter(conn),
		q:           NewPriorityQueue(DefaultPeerSendQueueSize, DefaultSendQueueMaxPriority),
		incomming:   incomming,
		timestamp:   time.Now(),
		pool:        NewTimestampPool(DefaultPeerPoolExpireSecond + 1),
		close:       make(chan error),
		closeReason: make([]string, 0),
		closeErr:    make([]error, 0),
		onError:     func(err error, p *Peer, pkt *Packet) { p.CloseByError(err) },
		onClose:     func(p *Peer) {},
	}
	p.setPacketCbFunc(cbFunc)

	return p
}

func (p *Peer) ResetConn(conn net.Conn) {
	p.conn = conn
	p.reader.Reset(conn)
	p.writer.Reset(conn)
}

func (p *Peer) String() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("{id:%v, conn:%s, addr:%v, in:%v, channel:%v, role:%v, type:%v, rtt:%v, children:%d, nephews:%d}",
		p.id, p.ConnString(), p.netAddress, p.incomming, p.channel, p.role, p.connType, p.rtt.String(), len(p.children), p.nephews)
}
func (p *Peer) ConnString() string {
	if p == nil {
		return ""
	}
	if p.incomming {
		return fmt.Sprint(p.conn.LocalAddr(), "<-", p.conn.RemoteAddr())
	} else {
		return fmt.Sprint(p.conn.LocalAddr(), "->", p.conn.RemoteAddr())
	}
}

func (p *Peer) ID() module.PeerID {
	return p.id
}

func (p *Peer) NetAddress() NetAddress {
	return p.netAddress
}

func (p *Peer) setPacketCbFunc(cbFunc packetCbFunc) {
	p.onPacket = cbFunc
	if cbFunc != nil {
		p.once.Do(func() {
			go p.receiveRoutine()
			go p.sendRoutine()
		})
	}
}

func (p *Peer) setErrorCbFunc(cbFunc errorCbFunc) {
	p.onError = cbFunc
}

func (p *Peer) setCloseCbFunc(cbFunc closeCbFunc) {
	p.onClose = cbFunc
}

func (p *Peer) setRole(r PeerRoleFlag) {
	defer p.roleMtx.Unlock()
	p.roleMtx.Lock()
	p.role = r
}
func (p *Peer) getRole() PeerRoleFlag {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.role
}
func (p *Peer) compareRole(r PeerRoleFlag, equal bool) bool {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	if equal {
		return p.role == r
	}
	return p.role.Has(r)
}

func (p *Peer) _close(err error) {
	if cerr := p.conn.Close(); cerr == nil {
		p.closed = true
		if err != nil && !p.isCloseError(err) {
			log.Printf("Warning Peer[%s].Close by error %+v", p.ConnString(), err)
		}
		p.onClose(p)
		close(p.close)
	}
}

func (p *Peer) Close(reason string) {
	p.closeReason = append(p.closeReason, reason)
	p._close(nil)
}

func (p *Peer) CloseByError(err error) {
	p.closeErr = append(p.closeErr, err)
	p._close(err)
}

func (p *Peer) CloseInfo() string {
	reason := "reason:["
	for i, s := range p.closeReason {
		if i != 0 {
			reason += ","
		}
		reason += "\"" + s + "\""
	}
	reason += "],"
	closeErr := "closeErr:["
	for i, e := range p.closeErr {
		if i != 0 {
			reason += ","
		}
		if p.isCloseError(e) {
			closeErr += "CLOSED_ERR"
		}
		closeErr += fmt.Sprintf("{%T %v}", e, e)
	}
	closeErr += "]"
	return reason + closeErr
}

func (p *Peer) _recover() interface{} {
	if err := recover(); err != nil {
		log.Printf("Warning Peer[%s]._recover from %+v", p.ConnString(), err)
		p._close(fmt.Errorf("_recover from %+v", err))
		return err
	}
	return nil
}

func (p *Peer) isCloseError(err error) bool {
	if oe, ok := err.(*net.OpError); ok {
		// if se, ok := oe.Err.(syscall.Errno); ok {
		// 	return se == syscall.ECONNRESET || se == syscall.ECONNABORTED
		// }
		//referenced from golang.org/x/net/http2/server.go isClosedConnError
		if strings.Contains(oe.Err.Error(), "use of closed network connection") ||
			strings.Contains(oe.Err.Error(), "connection reset by peer") {
			return true
		}
	} else if err == io.EOF || err == io.ErrUnexpectedEOF { //half Close (recieved tcp close)
		return true
	}
	return false
}

func (p *Peer) isTemporaryError(err error) bool {
	if oe, ok := err.(*net.OpError); ok { //after p.conn.Close()
		// log.Printf("Peer.isTemporaryError OpError %+v %#v %#v %s", oe, oe, oe.Err, p.String())
		// if se, ok := oe.Err.(*os.SyscallError); ok {
		// 	log.Printf("Peer.isTemporaryError *os.SyscallError %+v %#v %#v %s", se, se.Err, se.Err, p.String())
		// }
		return oe.Temporary()
	} else if err == io.EOF || err == io.ErrUnexpectedEOF { //half Close (recieved tcp close)
		return false
	}
	return true
}

//receive from bufio.Reader, unmarshalling and peerToPeer.onPacket
func (p *Peer) receiveRoutine() {
	defer func() {
		if err := p._recover(); err == nil {
			p.Close("receiveRoutine finish")
		}
		// log.Println("Peer.receiveRoutine finish", p.String())
	}()
	for {
		pkt, h, err := p.reader.ReadPacket()
		if err != nil {
			r := p.isTemporaryError(err)
			// log.Printf("Peer.receiveRoutine Error isTemporary:{%v} error:{%+v} peer:%s", r, err, p.String())
			if !r {
				p.CloseByError(err)
				return
			}
			//TODO p.reader.Reset()
			p.onError(err, p, pkt)
			continue
		}
		if pkt.hashOfPacket != h.Sum64() {
			log.Println(p.id, "Peer", "receiveRoutine", "Drop, Invalid hash:", pkt.hashOfPacket, ",expected:", h.Sum64(), pkt.protocol, pkt.subProtocol)
			continue
		} else {
			pkt.sender = p.id
			p.pool.Put(pkt.hashOfPacket)
			if cbFunc := p.onPacket; cbFunc != nil {
				cbFunc(pkt, p)
			} else {
				log.Printf("Warning Peer[%s].onPacket in nil, Drop %s", p.ConnString(), pkt.String())
			}
		}
	}
}

func (p *Peer) sendDirect(pkt *Packet) error {
	defer p.mtx.Unlock()
	p.mtx.Lock()

	if err := p.conn.SetWriteDeadline(time.Now().Add(DefaultSendTimeout)); err != nil {
		return err
	} else if err := p.writer.WritePacket(pkt); err != nil {
		return err
	} else if err := p.writer.Flush(); err != nil {
		return err
	}
	return nil
}

func (p *Peer) sendRoutine() {
	// defer func() {
	// 	log.Println("Peer.sendRoutine end", p.String())
	// }()
Loop:
	for {
		select {
		case <-p.close:
			break Loop
		case <-p.q.Wait():
			for {
				ctx := p.q.Pop()
				if ctx == nil {
					break
				}
				pkt := ctx.Value(p2pContextKeyPacket).(*Packet)

				if pkt.hashOfPacket != 0 {
					p.pool.RemoveBefore(DefaultPeerPoolExpireSecond)
					if p.pool.Contains(pkt.hashOfPacket) {
						//TODO drop counting each (protocol,subProtocol)
						//log.Println(p.id, "Peer", "sendRoutine", "Drop, Duplicated by hash",p.ConnString(), pkt)
						continue
					}
				}

				if err := p.sendDirect(pkt); err != nil {
					r := p.isTemporaryError(err)
					// log.Printf("Peer.sendRoutine Error isTemporary:{%v} error:{%+v} peer:%s", r, err, p.String())
					if !r {
						p.CloseByError(err)
						return
					}
					//TODO p.writer.Reset()
					p.onError(err, p, pkt)
				}
				//log.Println(p.id, "Peer", "sendRoutine",p.connType, p.ConnString(), pkt)
				//p.pool.Put(pkt.hashOfPacket)
			}
		}
	}
}

func (p *Peer) isDuplicatedToSend(pkt *Packet) bool {
	if p.id.Equal(pkt.src) {
		//log.Println(p.id, "Peer", "send", "Drop, Duplicated by src",p.ConnString(), pkt)
		return true
	}
	if pkt.sender != nil && p.id.Equal(pkt.sender) {
		//log.Println(p.id, "Peer", "send", "Drop, Duplicated by sender",p.ConnString(), pkt)
		return true
	}

	return false
}

func (p *Peer) send(ctx context.Context) error {
	if p == nil || p.closed {
		return ErrNotAvailable
	}
	c := ctx.Value(p2pContextKeyCounter).(*Counter)
	c.peer++
	pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
	if p.isDuplicatedToSend(pkt) {
		c.duplicate++
		return ErrDuplicatedPacket
	}
	if ok := p.q.Push(ctx, pkt.priority); !ok {
		c.overflow++
		return ErrQueueOverflow
	}
	c.enqueue++
	return nil
}

func (p *Peer) sendPacket(pkt *Packet) error {
	if p == nil || p.closed {
		return ErrNotAvailable
	}
	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyCounter, &Counter{})
	return p.send(ctx)
}

const (
	peerIDSize = 20 //common.AddressIDBytes
)

type peerID struct {
	*common.Address
}

func NewPeerID(b []byte) module.PeerID {
	return &peerID{common.NewAccountAddress(b)}
}

func NewPeerIDFromAddress(a module.Address) module.PeerID {
	return NewPeerID(a.ID())
}

func NewPeerIDFromPublicKey(k *crypto.PublicKey) module.PeerID {
	return &peerID{common.NewAccountAddressFromPublicKey(k)}
}

func NewPeerIDFromString(s string) module.PeerID {
	a := common.NewAddressFromString(s)
	if a.IsContract() {
		panic("PeerId must be AccountAddress")
	}
	return &peerID{a}
}

func (pi *peerID) Copy(b []byte) {
	copy(b[:peerIDSize], pi.ID())
}

func (pi *peerID) Equal(a module.Address) bool {
	return a.Equal(pi.Address)
}

func (pi *peerID) String() string {
	return pi.Address.String()
}
