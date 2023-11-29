package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

type Peer struct {
	//
	conn         net.Conn
	reader       *PacketReader
	writer       *PacketWriter
	q            *PriorityQueue
	onPacket     packetCbFunc
	onClose      closeCbFunc
	cbMtx        sync.RWMutex
	timestamp    time.Time
	pool         *TimestampPool
	close        chan error
	closed       int32
	closeReason  []string
	closeErr     []error
	closeInfoMtx sync.RWMutex
	sendMtx      sync.Mutex
	once         sync.Once

	//properties
	id            module.PeerID
	idMtx         sync.RWMutex
	netAddress    NetAddress
	netAddressMtx sync.RWMutex
	peerAddress   PeerAddress
	dial          NetAddress
	in            bool
	channel       string
	channelMtx    sync.RWMutex
	connType      PeerConnectionType
	connTypeMtx   sync.RWMutex
	recvConnType  PeerConnectionType
	role          PeerRoleFlag
	roleMtx       sync.RWMutex
	recvRole      PeerRoleFlag
	conns         map[PeerConnectionType]*GenericSet[PeerAddress]
	pis           *ProtocolInfos
	pisMtx        sync.RWMutex
	attr          map[string]interface{}
	attrMtx       sync.RWMutex

	//
	secureKey *secureKey
	rtt       PeerRTT

	//log
	logger log.Logger

	//monitor
	mtr       *metric.NetworkMetric
	metricMtx sync.RWMutex
}

type packetCbFunc func(pkt *Packet, p *Peer)
type closeCbFunc func(p *Peer)

func newPeer(conn net.Conn, in bool, dial NetAddress, l log.Logger) *Peer {
	conns := map[PeerConnectionType]*GenericSet[PeerAddress]{
		p2pConnTypeChildren: NewGenericSet[PeerAddress](),
		p2pConnTypeNephew:   NewGenericSet[PeerAddress](),
		p2pConnTypeOther:    NewGenericSet[PeerAddress](),
	}
	return &Peer{
		conn:        conn,
		reader:      NewPacketReader(conn),
		writer:      NewPacketWriter(conn),
		q:           NewPriorityQueue(DefaultPeerSendQueueSize, DefaultSendQueueMaxPriority),
		in:          in,
		timestamp:   time.Now(),
		pool:        NewTimestampPool(DefaultPeerPoolExpireSecond + 1),
		close:       make(chan error),
		closeReason: make([]string, 0),
		closeErr:    make([]error, 0),
		conns:       conns,
		attr:        make(map[string]interface{}),
		dial:        dial,
		logger:      l,
	}
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
	return fmt.Sprintf("{id:%v, conn:%s, addr:%v, in:%v, channel:%v, role:%v, rrole:%v, type:%v, rtype:%v, rtt:%v, children:%d, nephews:%d}",
		p.ID(), p.ConnString(), p.NetAddress(), p.In(), p.Channel(), p.Role(), p.RecvRole(), p.ConnType(), p.RecvConnType(), p.rtt.String(), p.conns[p2pConnTypeChildren].Len(), p.conns[p2pConnTypeNephew].Len())
}
func (p *Peer) ConnString() string {
	if p == nil {
		return ""
	}
	if p.In() {
		return fmt.Sprint(p.conn.LocalAddr(), "<-", p.conn.RemoteAddr())
	} else {
		return fmt.Sprint(p.conn.LocalAddr(), "->", p.conn.RemoteAddr())
	}
}

func (p *Peer) In() bool {
	return p.in
}

func (p *Peer) DialNetAddress() NetAddress {
	return p.dial
}

func (p *Peer) setID(id module.PeerID) {
	p.idMtx.Lock()
	defer p.idMtx.Unlock()
	p.id = id
}

func (p *Peer) ID() module.PeerID {
	p.idMtx.Lock()
	defer p.idMtx.Unlock()
	return p.id
}

func (p *Peer) setNetAddress(na NetAddress) {
	p.netAddressMtx.Lock()
	defer p.netAddressMtx.Unlock()
	p.netAddress = na
	p.peerAddress = NewPeerAddress(p.id, na)
}

func (p *Peer) NetAddress() NetAddress {
	p.netAddressMtx.RLock()
	defer p.netAddressMtx.RUnlock()
	return p.netAddress
}

func (p *Peer) PeerAddress() PeerAddress {
	p.netAddressMtx.RLock()
	defer p.netAddressMtx.RUnlock()
	return p.peerAddress
}

func (p *Peer) setChannel(c string) {
	p.channelMtx.Lock()
	defer p.channelMtx.Unlock()
	p.channel = c
}

func (p *Peer) Channel() string {
	p.channelMtx.RLock()
	defer p.channelMtx.RUnlock()
	return p.channel
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

func (p *Peer) setConnType(ct PeerConnectionType) {
	p.connTypeMtx.Lock()
	defer p.connTypeMtx.Unlock()
	p.connType = ct
}

func (p *Peer) ConnType() PeerConnectionType {
	p.connTypeMtx.RLock()
	defer p.connTypeMtx.RUnlock()
	return p.connType
}

func (p *Peer) setRecvConnType(ct PeerConnectionType) {
	p.connTypeMtx.Lock()
	defer p.connTypeMtx.Unlock()
	p.recvConnType = ct
}

func (p *Peer) RecvConnType() PeerConnectionType {
	p.connTypeMtx.RLock()
	defer p.connTypeMtx.RUnlock()
	return p.recvConnType
}

func (p *Peer) setRole(r PeerRoleFlag) {
	defer p.roleMtx.Unlock()
	p.roleMtx.Lock()
	p.role = r
}
func (p *Peer) Role() PeerRoleFlag {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.role
}
func (p *Peer) HasRole(r PeerRoleFlag) bool {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.role.Has(r)
}
func (p *Peer) EqualsRole(r PeerRoleFlag) bool {
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
func (p *Peer) setRecvRole(r PeerRoleFlag) {
	defer p.roleMtx.Unlock()
	p.roleMtx.Lock()
	p.recvRole = r
}
func (p *Peer) RecvRole() PeerRoleFlag {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.recvRole
}
func (p *Peer) HasRecvRole(r PeerRoleFlag) bool {
	defer p.roleMtx.RUnlock()
	p.roleMtx.RLock()
	return p.recvRole.Has(r)
}

func (p *Peer) Conns(connType PeerConnectionType) *GenericSet[PeerAddress] {
	return p.conns[connType]
}

func (p *Peer) ResetConnsByNetAddresses(connType PeerConnectionType, l []NetAddress) {
	var tl []PeerAddress
	for _, v := range l {
		tl = append(tl, NewPeerAddress(nil, v))
	}
	p.conns[connType].Reset(tl...)
}

func (p *Peer) UpdateConns(connType PeerConnectionType, dr DiffResult[PeerAddress]) {
	if s, ok := p.conns[connType]; ok {
		s.SetByDiff(dr)
	}
}

func (p *Peer) _close() (err error) {
	if atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		if err = p.conn.Close(); err != nil {
			p.logger.Debugf("Peer[%s]._close err:%+v", p.ConnString(), err)
		}
		close(p.close)
		if cbFunc := p.getCloseCbFunc(); cbFunc != nil {
			cbFunc(p)
		}
	}
	return
}

func (p *Peer) IsClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *Peer) WaitClose() {
	if !p.IsClosed() {
		<-p.close
	}
}

func (p *Peer) addCloseReason(reason string) {
	p.closeInfoMtx.Lock()
	defer p.closeInfoMtx.Unlock()

	p.closeReason = append(p.closeReason, reason)
}

func (p *Peer) Close(reason string) error {
	p.addCloseReason(reason)
	return p._close()
}

func (p *Peer) addCloseError(err error) {
	p.closeInfoMtx.Lock()
	defer p.closeInfoMtx.Unlock()

	p.closeErr = append(p.closeErr, err)
}

func (p *Peer) CloseByError(err error) error {
	p.addCloseError(err)
	return p._close()
}

func (p *Peer) CloseInfo() string {
	p.closeInfoMtx.RLock()
	defer p.closeInfoMtx.RUnlock()

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
		if err := recover(); err != nil {
			p.logger.Warnf("Peer[%s].receiveRoutine recover from %+v\n %s", p.ConnString(), err, string(debug.Stack()))
			p.CloseByError(fmt.Errorf("recover from %+v", err))
		} else {
			p.Close("receiveRoutine finish")
		}
	}()
	for {
		pkt, err := p.reader.ReadPacket()
		if err != nil {
			r := p.isTemporaryError(err)
			p.logger.Tracef("Peer.receiveRoutine Error isTemporary:{%v} error:{%+v} peer:%s pkt:%s",
				r, err, p, pkt)
			if !r {
				p.CloseByError(err)
				return
			}
			continue
		}

		pkt.sender = p.ID()
		p.pool.Put(pkt.hashOfPacket)
		p.getMetric().OnRecv(pkt.dest, pkt.ttl, pkt.extendInfo.hint(), pkt.protocol.Uint16(), pkt.lengthOfPayload)
		if cbFunc := p.getPacketCbFunc(); cbFunc != nil {
			cbFunc(pkt, p)
		} else {
			p.logger.Infof("Peer[%s].onPacket in nil, Drop %s", p.ConnString(), pkt.String())
		}
	}
}

func (p *Peer) sendDirect(pkt *Packet) error {
	defer p.sendMtx.Unlock()
	p.sendMtx.Lock()

	if err := p.conn.SetWriteDeadline(time.Now().Add(DefaultSendTimeout)); err != nil {
		return err
	} else if err := p.writer.WritePacket(pkt); err != nil {
		return err
	}
	return nil
}

func (p *Peer) sendRoutine() {
	secondTick := time.NewTicker(time.Second)
	defer secondTick.Stop()
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
					p.logger.Tracef("Peer.sendRoutine Error isTemporary:{%v} error:{%+v} peer:%s pkt:%s",
						r, err, p, pkt)
					p.CloseByError(err)
					return
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
	if p.ID().Equal(pkt.src) {
		return true
	}
	if !pkt.forceSend {
		if pkt.sender != nil && p.ID().Equal(pkt.sender) {
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

func (p *Peer) HasCloseError(err error) bool {
	p.closeInfoMtx.RLock()
	defer p.closeInfoMtx.RUnlock()

	for _, cerr := range p.closeErr {
		if err != nil && (cerr == err || cerr.Error() == err.Error()) {
			return true
		}
	}
	return false
}

func (p *Peer) ProtocolInfos() *ProtocolInfos {
	p.pisMtx.RLock()
	defer p.pisMtx.RUnlock()

	return p.pis
}

func (p *Peer) setProtocolInfos(pis *ProtocolInfos) {
	p.pisMtx.Lock()
	defer p.pisMtx.Unlock()

	p.pis = pis
}

func (p *Peer) GetAttr(k string) (interface{}, bool) {
	p.attrMtx.RLock()
	defer p.attrMtx.RUnlock()
	v, ok := p.attr[k]
	return v, ok
}

func (p *Peer) PutAttr(k string, v interface{}) {
	p.attrMtx.Lock()
	defer p.attrMtx.Unlock()
	p.attr[k] = v
}

func (p *Peer) RemoveAttr(k string) {
	p.attrMtx.Lock()
	defer p.attrMtx.Unlock()
	if _, ok := p.attr[k]; ok {
		delete(p.attr, k)
	}
}

func (p *Peer) EqualsAttr(k string, v interface{}) bool {
	p.attrMtx.RLock()
	defer p.attrMtx.RUnlock()
	ov, ok := p.attr[k]
	return ok && ov == v
}

func (p *Peer) GetAndHandleAttr(k string, h func(v interface{}, exists bool) bool) (interface{}, bool) {
	p.attrMtx.Lock()
	defer p.attrMtx.Unlock()
	v, exists := p.attr[k]
	r := h(v, exists)
	return v, r
}

type NetAddress string

func (na NetAddress) Validate() error {
	_, port, err := net.SplitHostPort(string(na))
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
	t    *time.Timer
	mtx  sync.RWMutex
}

func NewPeerRTT() *PeerRTT {
	return &PeerRTT{}
}

func (r *PeerRTT) Start() {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.t != nil {
		//ignore
		r.t.Stop()
		r.t = nil
	}
	r.st = time.Now()
}

func (r *PeerRTT) StartWithAfterFunc(to time.Duration, f func()) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.t != nil {
		//ignore
		r.t.Stop()
		r.t = nil
	}
	r.t = time.AfterFunc(to, f)

	r.st = time.Now()
}

func (r *PeerRTT) Stop() time.Duration {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if r.t != nil {
		r.t.Stop()
		r.t = nil
	}

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
	return r.last
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

func (r *PeerRTT) Value() (time.Duration, time.Duration) {
	r.mtx.RLock()
	defer r.mtx.RUnlock()
	return r.last, r.avg
}

func (r *PeerRTT) String() string {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	return fmt.Sprintf("{last:%v,avg:%v}", r.last.String(), r.avg.String())
}

const (
	p2pRoleNone = PeerRoleFlag(module.RoleNormal)
	p2pRoleSeed = PeerRoleFlag(module.RoleSeed)
	p2pRoleRoot = PeerRoleFlag(module.RoleValidator)
)

//PeerRoleFlag as BitFlag MSB[_,_,_,_,_,_,Root,Seed]LSB
type PeerRoleFlag byte

func (pr PeerRoleFlag) Has(o PeerRoleFlag) bool {
	return pr&o == o
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
		roles = append(roles, module.RoleSeed)
	}
	if pr.Has(p2pRoleRoot) {
		roles = append(roles, module.RoleValidator)
	}
	return roles
}

func NewPeerRoleFlag(roles ...module.Role) PeerRoleFlag {
	var prf PeerRoleFlag
	for _, role := range roles {
		if role < module.RoleReserved {
			prf.SetFlag(PeerRoleFlag(role))
		}
	}
	return prf
}

const (
	p2pConnTypeNone PeerConnectionType = iota
	p2pConnTypeParent
	p2pConnTypeChildren
	p2pConnTypeUncle
	p2pConnTypeNephew
	p2pConnTypeFriend
	p2pConnTypeOther
	p2pConnTypeReserved
)

var (
	strPeerConnectionType = []string{
		"Orphanage",
		"Parent",
		"Children",
		"Uncle",
		"Nephew",
		"Friend",
		"Other",
	}
	joinPeerConnectionTypes = []PeerConnectionType{
		p2pConnTypeParent, p2pConnTypeUncle,
		p2pConnTypeChildren, p2pConnTypeNephew,
		p2pConnTypeOther, p2pConnTypeFriend,
	}
)

type PeerConnectionType byte

type PeerAddress string

func (pa PeerAddress) PeerID() module.PeerID {
	id, _, _ := pa.parse()
	return id
}
func (pa PeerAddress) NetAddress() NetAddress {
	_, na, _ := pa.parse()
	return na
}
func (pa PeerAddress) Validate() error {
	_, na, err := pa.parse()
	if err != nil {
		return err
	}
	return na.Validate()
}

func (pa PeerAddress) parse() (module.PeerID, NetAddress, error) {
	s := strings.Split(string(pa), "@")
	if len(s) != 2 {
		return nil, "", errors.Errorf("invalid PeerAddress:%s", pa)
	}
	id, err := NewPeerIDFromString(s[0])
	if err != nil {
		return nil, "", err
	}
	return id, NetAddress(s[1]), nil
}

func NewPeerAddress(id module.PeerID, na NetAddress) PeerAddress {
	s := ""
	if id != nil {
		s = id.String()
	}
	return PeerAddress(fmt.Sprintf("%s@%s", s, na))
}
