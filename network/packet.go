package network

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	packetHeaderSize = 10 + peerIDSize
	packetFooterSize = 10
)

//srcPeerId, castType, destInfo, TTL(0:unlimited)
type Packet struct {
	//header
	protocol        module.ProtocolInfo //2byte
	subProtocol     module.ProtocolInfo //2byte
	src             module.PeerID       //20byte
	dest            byte
	ttl             byte
	lengthOfPayload uint32 //4byte
	//footer
	hashOfPacket uint64 //8byte
	extendInfo   packetExtendInfo
	//bytes
	header  []byte
	payload []byte
	footer  []byte
	ext     []byte
	//Transient fields
	sender    module.PeerID //20byte
	destPeer  module.PeerID //20byte
	priority  uint8
	timestamp time.Time
	forceSend bool
	mtx       sync.RWMutex
}

type packetDestInfo uint16

const (
	p2pDestAny  = 0x00
	p2pDestSeed = byte(module.RoleSeed)
	p2pDestRoot = byte(module.RoleValidator)
	p2pDestPeer = 0xFF
)

func newPacketDestInfo(dest byte, ttl byte) packetDestInfo {
	return packetDestInfo(int(dest)<<8 | int(ttl))
}
func newPacketDestInfoFrom(b []byte) packetDestInfo {
	return packetDestInfo(binary.BigEndian.Uint16(b[:2]))
}
func (i packetDestInfo) dest() byte {
	return byte(i >> 8)
}
func (i packetDestInfo) ttl() byte {
	return byte(i >> 8)
}
func (i packetDestInfo) String() string {
	return fmt.Sprintf("{%#04x}", uint16(i))
}

type packetExtendInfo uint16

const (
	packetExtendMaxHint = 0x3F   // (1<<6)-1
	packetExtendMaxLen  = 0x03FF // (1<<10)-1
)

func newPacketExtendInfo(hint byte, len int) packetExtendInfo {
	return packetExtendInfo(int(hint)<<10 | int(len&packetExtendMaxLen))
}
func newPacketExtendInfoFrom(b []byte) packetExtendInfo {
	return packetExtendInfo(binary.BigEndian.Uint16(b[:2]))
}
func (i packetExtendInfo) len() int {
	l := i & packetExtendMaxLen
	return int(l)
}
func (i packetExtendInfo) hint() byte {
	h := i >> 10 & packetExtendMaxHint
	return byte(h)
}
func (i packetExtendInfo) String() string {
	return fmt.Sprintf("{hint:%d,len:%d}", i.hint(), i.len())
}

func NewPacket(pi module.ProtocolInfo, spi module.ProtocolInfo, payload []byte) *Packet {
	lengthOfPayload := len(payload)
	if lengthOfPayload > DefaultPacketPayloadMax {
		lengthOfPayload = DefaultPacketPayloadMax
	}
	return &Packet{
		protocol:        pi,
		subProtocol:     spi,
		lengthOfPayload: uint32(lengthOfPayload),
		payload:         payload[:lengthOfPayload],
		timestamp:       time.Now(),
	}
}

func newPacket(pi module.ProtocolInfo, spi module.ProtocolInfo, payload []byte, src module.PeerID) *Packet {
	pkt := NewPacket(pi, spi, payload)
	pkt.dest = p2pDestPeer
	pkt.ttl = 1
	pkt.src = src
	pkt.forceSend = true
	return pkt
}

func (p *Packet) String() string {
	return fmt.Sprintf("{pi:%#04x,subPi:%#04x,src:%v,dest:%#x,ttl:%d,len:%v,footer:%#x,ext:%v,sender:%v}",
		p.protocol.Uint16(),
		p.subProtocol.Uint16(),
		p.src,
		p.dest,
		p.ttl,
		p.lengthOfPayload,
		p.hashOfPacket,
		p.extendInfo,
		p.sender)
}

func (p *Packet) Len() int64 {
	return int64(len(p.header)) + int64(len(p.payload)) + int64(len(p.footer)) + int64(len(p.ext))
}

func (p *Packet) _read(r io.Reader, n int) ([]byte, int, error) {
	if n < 0 {
		return nil, 0, fmt.Errorf("invalid n:%d", n)
	}
	b := make([]byte, n)
	if n == 0 {
		return b, 0, nil
	}
	rn := 0
	for {
		tn, err := r.Read(b[rn:])
		if rn += tn; err != nil {
			return nil, rn, err
		}
		if rn >= n {
			break
		}
	}
	return b, rn, nil
}

func (p *Packet) WriteTo(w io.Writer) (n int64, err error) {
	if err = p.updateHash(false); err != nil {
		return
	}

	var tn int
	tn, err = w.Write(p.headerToBytes(false))
	if n += int64(tn); err != nil {
		return
	}
	tn, err = w.Write(p.payload[:p.lengthOfPayload])
	if n += int64(tn); err != nil {
		return
	}
	tn, err = w.Write(p.footerToBytes(false))
	if n += int64(tn); err != nil {
		return
	}
	if p.extendInfo.len() > 0 {
		tn, err = w.Write(p.ext[:p.extendInfo.len()])
		if n += int64(tn); err != nil {
			return
		}
	}
	return
}

func (p *Packet) updateHash(force bool) error {
	if p.hashOfPacket == 0 || force {
		h, err := p._hash(force)
		if err != nil {
			return err
		}
		p.hashOfPacket = h.Sum64()
	}
	return nil
}

func (p *Packet) _hash(force bool) (hash.Hash64, error) {
	h := fnv.New64a()
	if _, err := h.Write(p.headerToBytes(force)); err != nil {
		return nil, err
	}
	if _, err := h.Write(p.payload[:p.lengthOfPayload]); err != nil {
		return nil, err
	}
	return h, nil
}

func (p *Packet) headerToBytes(force bool) []byte {
	if force || p.header == nil {
		p.header = make([]byte, packetHeaderSize)
		tb := p.header[:]
		binary.BigEndian.PutUint16(tb[:2], p.protocol.Uint16())
		tb = tb[2:]
		binary.BigEndian.PutUint16(tb[:2], p.subProtocol.Uint16())
		tb = tb[2:]
		copy(tb[:peerIDSize], p.src.Bytes())
		tb = tb[peerIDSize:]
		tb[0] = p.dest
		tb = tb[1:]
		tb[0] = p.ttl
		tb = tb[1:]
		binary.BigEndian.PutUint32(tb[:4], p.lengthOfPayload)
		tb = tb[4:]
	}
	return p.header[:]
}

func (p *Packet) footerToBytes(force bool) []byte {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if force || p.footer == nil {
		footer := make([]byte, packetFooterSize)
		tb := footer[:]
		binary.BigEndian.PutUint64(tb[:8], p.hashOfPacket)
		tb = tb[8:]
		binary.BigEndian.PutUint16(tb[:2], uint16(p.extendInfo))
		tb = tb[2:]
		//lock
		p.footer = footer[:]
		//unlock
	}
	return p.footer[:]
}

func (p *Packet) ReadFrom(r io.Reader) (n int64, err error) {
	var b []byte
	var tn int
	b, tn, err = p._read(r, packetHeaderSize)
	if n += int64(tn); err != nil {
		return
	}
	if _, err = p.setHeader(b); err != nil {
		return
	}

	p.payload, tn, err = p._read(r, int(p.lengthOfPayload))
	if n += int64(tn); err != nil {
		return
	}

	b, tn, err = p._read(r, packetFooterSize)
	if n += int64(tn); err != nil {
		return
	}
	if _, err = p.setFooter(b); err != nil {
		return
	}

	if p.extendInfo.len() > 0 {
		p.ext, tn, err = p._read(r, int(p.extendInfo.len()))
		if n += int64(tn); err != nil {
			return
		}
	}

	h, err := p._hash(false)
	if err != nil {
		return
	}
	if h.Sum64() != p.hashOfPacket {
		err = fmt.Errorf("invalid hashOfPacket %v expected:%#x", p, h.Sum64())
		return
	}
	return
}

func (p *Packet) setHeader(b []byte) ([]byte, error) {
	if len(b) < packetHeaderSize {
		//io.ErrShortBuffer
		return b, fmt.Errorf("short buffer")
	}
	p.header = b[:packetHeaderSize]
	tb := p.header[:]
	p.protocol = module.ProtocolInfo(binary.BigEndian.Uint16(tb[:2]))
	tb = tb[2:]
	p.subProtocol = module.ProtocolInfo(binary.BigEndian.Uint16(tb[:2]))
	tb = tb[2:]
	p.src = NewPeerID(tb[:peerIDSize])
	tb = tb[peerIDSize:]
	p.dest = tb[0]
	tb = tb[1:]
	p.ttl = tb[0]
	tb = tb[1:]
	p.lengthOfPayload = binary.BigEndian.Uint32(tb[:4])
	tb = tb[4:]
	if p.lengthOfPayload > DefaultPacketPayloadMax {
		return b[packetHeaderSize:], fmt.Errorf("invalid lengthOfPayload")
	}
	return b[packetHeaderSize:], nil
}

func (p *Packet) setFooter(b []byte) ([]byte, error) {
	if len(b) < packetFooterSize {
		//io.ErrShortBuffer
		return b, fmt.Errorf("short buffer")
	}
	p.footer = b[:packetFooterSize]
	tb := p.footer[:]
	p.hashOfPacket = binary.BigEndian.Uint64(tb[:8])
	tb = tb[8:]
	p.extendInfo = newPacketExtendInfoFrom(tb[:2])
	tb = tb[2:]
	return b[packetFooterSize:], nil
}

type PacketReader struct {
	*bufio.Reader
	rd   io.Reader
	pkt  *Packet
	hash hash.Hash64
}

// NewPacketReader returns a new PacketReader whose buffer has the default size.
func NewPacketReader(rd io.Reader) *PacketReader {
	return &PacketReader{Reader: bufio.NewReaderSize(rd, DefaultPacketBufferSize), rd: rd}
}

func (pr *PacketReader) Reset(rd io.Reader) {
	pr.rd = rd
	pr.Reader.Reset(pr.rd)
}

func (pr *PacketReader) ReadPacket() (pkt *Packet, e error) {
	pkt = &Packet{}
	_, err := pkt.ReadFrom(pr)
	if err != nil {
		e = err
		return
	}
	return
}

type PacketWriter struct {
	*bufio.Writer
	wr  io.Writer
	mtx sync.Mutex
}

func NewPacketWriter(w io.Writer) *PacketWriter {
	return &PacketWriter{Writer: bufio.NewWriterSize(w, DefaultPacketBufferSize), wr: w}
}

func (pw *PacketWriter) Reset(wr io.Writer) {
	pw.mtx.Lock()
	defer pw.mtx.Unlock()

	pw.wr = wr
	pw.Writer.Reset(pw.wr)
}

func (pw *PacketWriter) WritePacket(pkt *Packet) error {
	_, err := pkt.WriteTo(pw)
	if err != nil {
		return err
	}
	if pw.Buffered() > 0 {
		return pw.Flush()
	}
	return nil
}

func (pw *PacketWriter) Write(b []byte) (int, error) {
	wn := 0
	re := 0
	for {
		n, err := pw.Writer.Write(b[wn:])
		wn += n
		if err != nil && err == io.ErrShortWrite && re < DefaultPacketRewriteLimit {
			re++
			time.Sleep(DefaultPacketRewriteDelay)
			continue
		} else {
			return wn, err
		}
	}
}

func (pw *PacketWriter) Flush() error {
	pw.mtx.Lock()
	defer pw.mtx.Unlock()

	re := 0
	for {
		err := pw.Writer.Flush()
		if err != nil && err == io.ErrShortWrite && re < DefaultPacketRewriteLimit {
			re++
			time.Sleep(DefaultPacketRewriteDelay)
			continue
		} else {
			return err
		}
	}
}
