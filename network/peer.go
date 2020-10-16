package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

const isLoggingPacket = false

type Peer struct {
	id         module.PeerID
	netAddress NetAddress
	secureKey  *secureKey
	//
	conn         net.Conn
	reader       *PacketReader
	writer       *PacketWriter
	q            *PriorityQueue
	onPacket     packetCbFunc
	onError      errorCbFunc
	onClose      closeCbFunc
	cbMtx        sync.RWMutex
	timestamp    time.Time
	pool         *TimestampPool
	close        chan error
	closed       int32
	closeReason  []string
	closeErr     []error
	closeInfoMtx sync.RWMutex
	mtx          sync.Mutex
	once         sync.Once
	//
	incomming bool
	channel   string
	rtt       PeerRTT
	connType  PeerConnectionType
	role      PeerRoleFlag
	roleMtx   sync.RWMutex
	children  *NetAddressSet
	nephews   int32
	//
	last context.Context

	//log
	logger log.Logger

	//monitor
	mtr       *metric.NetworkMetric
	metricMtx sync.RWMutex
}

type packetCbFunc func(pkt *Packet, p *Peer)
type errorCbFunc func(err error, p *Peer, pkt *Packet)
type closeCbFunc func(p *Peer)

type NetAddress string
func (na NetAddress) Validate() error {
	_,port,err := net.SplitHostPort(string(na))
	if err != nil {
		return err
	}
	if i, err := strconv.ParseInt(port, 10, 64); err != nil {
		return err
	} else if i < 1 || i > 65535 {
		return fmt.Errorf("invalid port %s", port)
	}
	return nil
}

type PeerRTT struct {
	last time.Duration
	avg  time.Duration
	st   time.Time
	et   time.Time
	mtx  sync.RWMutex
}

func NewPeerRTT() *PeerRTT {
	return &PeerRTT{}
}

func (r *PeerRTT) Start() time.Time {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.st = time.Now()
	return r.st
}

func (r *PeerRTT) Stop() time.Time {
	r.mtx.Lock()
	defer r.mtx.Unlock()

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
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	fv := float64(r.last) / float64(d)
	return fv
}

func (r *PeerRTT) Avg(d time.Duration) float64 {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	fv := float64(r.avg) / float64(d)
	return fv
}

func (r *PeerRTT) String() string {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	return fmt.Sprintf("{last:%v,avg:%v}", r.last.String(), r.avg.String())
}

const (
	p2pRoleNone     = 0x00
	p2pRoleSeed     = 0x01
	p2pRoleRoot     = 0x02
	p2pRoleRootSeed = 0x03
)

//PeerRoleFlag as BitFlag MSB[_,_,_,_,_,_,Root,Seed]LSB
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
func (pr *PeerRoleFlag) ToRoles() []module.Role {
	roles := make([]module.Role, 0)
	if pr.Has(p2pRoleSeed) {
		roles = append(roles, module.ROLE_SEED)
	}
	if pr.Has(p2pRoleRoot) {
		roles = append(roles, module.ROLE_VALIDATOR)
	}
	return roles
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
	defaultOnError = func(err error, p *Peer, pkt *Packet) { p.CloseByError(err) }
	defaultOnClose = func(p *Peer) {}
)

type PeerConnectionType byte

func newPeer(conn net.Conn, cbFunc packetCbFunc, incomming bool, l log.Logger) *Peer {
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
		onError:     defaultOnError,
		onClose:     defaultOnClose,
		children:    NewNetAddressSet(),
	}
	p.logger = l.WithFields(log.Fields{"peer": p.id})
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
		p.id, p.ConnString(), p.netAddress, p.incomming, p.channel, p.getRole(), p.connType, p.rtt.String(), p.children.Len(),
		atomic.LoadInt32(&p.nephews))
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
	p.cbMtx.Lock()
	defer p.cbMtx.Unlock()

	p.onPacket = cbFunc
	if cbFunc != nil {
		p.once.Do(func() {
			go p.receiveRoutine()
			go p.sendRoutine()
		})
	}
}

func (p *Peer) getPacketCbFunc() packetCbFunc {
	p.cbMtx.RLock()
	defer p.cbMtx.RUnlock()

	return p.onPacket
}

func (p *Peer) setErrorCbFunc(cbFunc errorCbFunc) {
	p.cbMtx.Lock()
	defer p.cbMtx.Unlock()

	p.onError = cbFunc
}

func (p *Peer) getErrorCbFunc() errorCbFunc {
	p.cbMtx.RLock()
	defer p.cbMtx.RUnlock()

	return p.onError
}

func (p *Peer) setCloseCbFunc(cbFunc closeCbFunc) {
	p.cbMtx.Lock()
	defer p.cbMtx.Unlock()

	p.onClose = cbFunc
}

func (p *Peer) getCloseCbFunc() closeCbFunc {
	p.cbMtx.RLock()
	defer p.cbMtx.RUnlock()

	return p.onClose
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
func (p *Peer) hasRole(r PeerRoleFlag) bool {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.role.Has(r)
}
func (p *Peer) equalRole(r PeerRoleFlag) bool {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.role == r
}
func (p *Peer) addRole(r PeerRoleFlag) {
	defer p.roleMtx.Unlock()
	p.roleMtx.Lock()
	p.role.SetFlag(r)
}
func (p *Peer) removeRole(r PeerRoleFlag) {
	defer p.roleMtx.Unlock()
	p.roleMtx.Lock()
	p.role.UnSetFlag(r)
}

func (p *Peer) _close() (err error) {
	if atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		if err = p.conn.Close(); err != nil {
			p.logger.Debugf("Peer[%s]._close err:%+v", p.ConnString(), err)
		}
		close(p.close)
		if cbFunc := p.getCloseCbFunc(); cbFunc != nil {
			cbFunc(p)
		} else {
			defaultOnClose(p)
		}
	}
	return
}

func (p *Peer) IsClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *Peer) Close(reason string) error {
	p.closeInfoMtx.Lock()
	defer p.closeInfoMtx.Unlock()

	p.closeReason = append(p.closeReason, reason)
	return p._close()
}

func (p *Peer) CloseByError(err error) error {
	p.closeInfoMtx.Lock()
	defer p.closeInfoMtx.Unlock()

	p.closeErr = append(p.closeErr, err)
	return p._close()
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
			closeErr += ","
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
		p.logger.Infof("Peer[%s]._recover from %+v", p.ConnString(), err)
		p.CloseByError(fmt.Errorf("_recover from %+v", err))
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
	}
	return false
}

//receive from bufio.Reader, unmarshalling and peerToPeer.onPacket
func (p *Peer) receiveRoutine() {
	defer func() {
		if err := p._recover(); err == nil {
			p.Close("receiveRoutine finish")
		}
	}()
	for {
		pkt, err := p.reader.ReadPacket()
		if err != nil {
			r := p.isTemporaryError(err)
			p.logger.Tracef("Peer.receiveRoutine Error isTemporary:{%v} error:{%+v} peer:%s", r, err, p.String())
			if !r {
				p.CloseByError(err)
				return
			}
			if cbFunc := p.getErrorCbFunc(); cbFunc != nil {
				cbFunc(err, p, pkt)
			} else {
				defaultOnError(err, p, pkt)
			}
			continue
		}

		pkt.sender = p.id
		p.pool.Put(pkt.hashOfPacket)
		p.getMetric().OnRecv(pkt.dest, pkt.ttl, pkt.extendInfo.hint(), pkt.protocol.Uint16(), pkt.lengthOfPayload)
		//TODO peer.packet_dump
		if isLoggingPacket {
			log.Println(p.id, "Peer", "receiveRoutine", p.connType, p.ConnString(), pkt)
		}
		if cbFunc := p.getPacketCbFunc(); cbFunc != nil {
			cbFunc(pkt, p)
		} else {
			p.logger.Infof("Peer[%s].onPacket in nil, Drop %s", p.ConnString(), pkt.String())
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
	secondTick := time.NewTicker(time.Second)
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
				if err := p.sendDirect(pkt); err != nil {
					r := p.isTemporaryError(err)
					p.logger.Tracef("Peer.sendRoutine Error isTemporary:{%v} error:{%+v} peer:%s", r, err, p.String())
					if !r {
						p.CloseByError(err)
						return
					}
					if cbFunc := p.getErrorCbFunc(); cbFunc != nil {
						cbFunc(err, p, pkt)
					} else {
						defaultOnError(err, p, pkt)
					}
				}
				//TODO peer.packet_dump
				if isLoggingPacket {
					log.Println(p.id, "Peer", "sendRoutine", p.connType, p.ConnString(), pkt)
				}
				p.pool.Put(pkt.hashOfPacket)
				p.getMetric().OnSend(pkt.dest, pkt.ttl, pkt.extendInfo.hint(), pkt.protocol.Uint16(), pkt.lengthOfPayload)
			}
		case <-secondTick.C:
			p.pool.RemoveBefore(DefaultPeerPoolExpireSecond)
		}
	}
}

func (p *Peer) isDuplicatedToSend(pkt *Packet) bool {
	if p.id.Equal(pkt.src) {
		return true
	}
	if !pkt.forceSend {
		if pkt.sender != nil && p.id.Equal(pkt.sender) {
			return true
		}
		if _ = pkt.updateHash(false); p.pool.Contains(pkt.hashOfPacket) {
			return true
		}
	}
	return false
}

func (p *Peer) send(ctx context.Context) error {
	if p == nil || p.IsClosed() {
		return ErrNotAvailable
	}
	c := ctx.Value(p2pContextKeyCounter).(*Counter)
	c.peer++
	pkt := ctx.Value(p2pContextKeyPacket).(*Packet)
	//TODO dequeue 전에 peer.send가 호출되면 duplication check가 되지 않음.
	if p.isDuplicatedToSend(pkt) {
		c.duplicate++
		return ErrDuplicatedPacket
	}
	if ok := p.q.Push(ctx, int(pkt.priority)); !ok {
		c.overflow++
		return ErrQueueOverflow
	}
	c.enqueue++
	return nil
}

func (p *Peer) sendPacket(pkt *Packet) error {
	if p == nil || p.IsClosed() {
		return ErrNotAvailable
	}
	ctx := context.WithValue(context.Background(), p2pContextKeyPacket, pkt)
	ctx = context.WithValue(ctx, p2pContextKeyCounter, &Counter{})
	return p.send(ctx)
}

func (p *Peer) setMetric(nm *metric.NetworkMetric) {
	p.metricMtx.Lock()
	defer p.metricMtx.Unlock()
	p.mtr = nm
}

func (p *Peer) getMetric() *metric.NetworkMetric {
	p.metricMtx.RLock()
	defer p.metricMtx.RUnlock()
	return p.mtr
}
