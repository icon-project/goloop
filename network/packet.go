package network

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"log"
	"sync"
	"time"

	"github.com/icon-project/goloop/module"
)

const (
	packetHeaderSize = 10 + peerIDSize
	packetFooterSize = 8
)

//srcPeerId, castType, destInfo, TTL(0:unlimited)
type Packet struct {
	protocol        protocolInfo  //2byte
	subProtocol     protocolInfo  //2byte
	src             module.PeerID //20byte
	dest            byte
	ttl             byte
	lengthOfpayload uint32 //4byte
	hashOfPacket    uint64 //8byte
	//
	header  []byte
	payload []byte
	footer  []byte
	//Transient fields
	sender    module.PeerID //20byte
	destPeer  module.PeerID //20byte
	priority  uint8
	timestamp time.Time
	forceSend bool
}

const (
	p2pDestAny       = 0x00
	p2pDestPeerGroup = 0x08
	p2pDestPeer      = 0xFF
)

func NewPacket(pi protocolInfo, spi protocolInfo, payload []byte) *Packet {
	return &Packet{
		protocol:        pi,
		subProtocol:     spi,
		lengthOfpayload: uint32(len(payload)),
		payload:         payload[:],
		timestamp:       time.Now(),
	}
}

func newPacket(spi protocolInfo, payload []byte, src module.PeerID) *Packet {
	pkt := NewPacket(PROTO_CONTOL, spi, payload)
	pkt.dest = p2pDestPeer
	pkt.src = src
	pkt.forceSend = true
	return pkt
}

func (p *Packet) String() string {
	return fmt.Sprintf("{pi:%#04x,subPi:%#04x,src:%v,dest:%#x,ttl:%d,len:%v,hash:%#x,sender:%v}",
		p.protocol.Uint16(),
		p.subProtocol.Uint16(),
		p.src,
		p.dest,
		p.ttl,
		p.lengthOfpayload,
		p.hashOfPacket,
		p.sender)
}

func (p *Packet) _read(r io.Reader, n int) ([]byte, int, error) {
	b := make([]byte, n)
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
	tn, err = w.Write(p.payload[:p.lengthOfpayload])
	if n += int64(tn); err != nil {
		return
	}
	tn, err = w.Write(p.footerToBytes(false))
	if n += int64(tn); err != nil {
		return
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
	if _, err := h.Write(p.payload[:p.lengthOfpayload]); err != nil {
		return nil, err
	}
	return h, nil
}

func (p *Packet) headerToBytes(force bool) []byte {
	if p.header == nil || force {
		p.header = make([]byte, packetHeaderSize)
		tb := p.header[:]
		p.protocol.Copy(tb[:2])
		tb = tb[2:]
		p.subProtocol.Copy(tb[:2])
		tb = tb[2:]
		p.src.Copy(tb[:peerIDSize])
		tb = tb[peerIDSize:]
		tb[0] = p.dest
		tb = tb[1:]
		tb[0] = p.ttl
		tb = tb[1:]
		binary.BigEndian.PutUint32(tb[:4], p.lengthOfpayload)
		tb = tb[4:]
	}
	return p.header[:]
}

func (p *Packet) footerToBytes(force bool) []byte {
	if p.footer == nil || force {
		p.footer = make([]byte, packetFooterSize)
		tb := p.footer[:]
		binary.BigEndian.PutUint64(tb[:8], p.hashOfPacket)
		tb = tb[8:]
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

	p.payload, tn, err = p._read(r, int(p.lengthOfpayload))
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

	h, err := p._hash(false)
	if err != nil {
		return
	}
	if h.Sum64() != p.hashOfPacket {
		err = fmt.Errorf("invalid hashOfPacket")
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
	p.protocol = newProtocolInfoFrom(tb[:2])
	tb = tb[2:]
	p.subProtocol = newProtocolInfoFrom(tb[:2])
	tb = tb[2:]
	p.src = NewPeerID(tb[:peerIDSize])
	tb = tb[peerIDSize:]
	p.dest = tb[0]
	tb = tb[1:]
	p.ttl = tb[0]
	tb = tb[1:]
	p.lengthOfpayload = binary.BigEndian.Uint32(tb[:4])
	tb = tb[4:]
	if p.lengthOfpayload > DefaultPacketPayloadMax {
		return b[packetHeaderSize:], fmt.Errorf("invalid lengthOfpayload")
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
	return b[packetFooterSize:], nil
}

type PacketReader struct {
	*bufio.Reader
	rd   io.Reader
	pkt  *Packet
	hash hash.Hash64
}

// NewReader returns a new Reader whose buffer has the default size.
func NewPacketReader(rd io.Reader) *PacketReader {
	return &PacketReader{Reader: bufio.NewReaderSize(rd, DefaultPacketBufferSize), rd: rd}
}

func (pr *PacketReader) _read(n int) ([]byte, error) {
	b := make([]byte, n)
	rn := 0
	for {
		tn, err := pr.Reader.Read(b[rn:])
		if err != nil {
			return nil, err
		}
		rn += tn
		if rn >= n {
			break
		}
	}
	return b, nil
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
	wr io.Writer
}

func NewPacketWriter(w io.Writer) *PacketWriter {
	return &PacketWriter{Writer: bufio.NewWriterSize(w, DefaultPacketBufferSize), wr: w}
}

func (pw *PacketWriter) Reset(wr io.Writer) {
	pw.wr = wr
	pw.Writer.Reset(pw.wr)
}

func (pw *PacketWriter) WritePacket(pkt *Packet) error {
	_, err := pkt.WriteTo(pw)
	if err != nil {
		log.Printf("PacketWriter.WritePacket fb %T %#v %s", err, err, err)
		return err
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
			log.Println("PacketWriter.Write io.ErrShortWrite", err)
			time.Sleep(DefaultPacketRewriteDelay)
			continue
		} else {
			return wn, err
		}
	}
}

func (pw *PacketWriter) Flush() error {
	re := 0
	for {
		err := pw.Writer.Flush()
		if err != nil && err == io.ErrShortWrite && re < DefaultPacketRewriteLimit {
			re++
			log.Println("PacketWriter.Flush io.ErrShortWrite", err)
			time.Sleep(DefaultPacketRewriteDelay)
			continue
		} else {
			return err
		}
	}
}

type PacketReadWriter struct {
	b    *bytes.Buffer
	rd   *PacketReader
	wr   *PacketWriter
	rpkt *Packet
	wpkt *Packet
	mtx  sync.RWMutex
}

func NewPacketReadWriter() *PacketReadWriter {
	b := bytes.NewBuffer(make([]byte, DefaultPacketBufferSize))
	b.Reset()
	return &PacketReadWriter{b: b, rd: NewPacketReader(b), wr: NewPacketWriter(b)}
}

func (prw *PacketReadWriter) WritePacket(pkt *Packet) error {
	defer prw.mtx.Unlock()
	prw.mtx.Lock()
	if err := prw.wr.WritePacket(pkt); err != nil {
		return err
	}
	if err := prw.wr.Flush(); err != nil {
		return err
	}
	prw.wpkt = pkt
	return nil
}

func (prw *PacketReadWriter) ReadPacket() (*Packet, error) {
	defer prw.mtx.RUnlock()
	prw.mtx.RLock()
	if prw.rpkt == nil {
		//(pkt *Packet, h hash.Hash64, e error)
		pkt, err := prw.rd.ReadPacket()
		if err != nil {
			return nil, err
		}
		prw.rpkt = pkt
	}
	return prw.rpkt, nil
}

func (prw *PacketReadWriter) Reset(rd io.Reader, wr io.Writer) {
	defer prw.mtx.Unlock()
	prw.mtx.Lock()
	prw.b.Reset()
	prw.rd.Reset(rd)
	prw.wr.Reset(wr)
	prw.rpkt = nil
	prw.wpkt = nil
}
