package network

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
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
	protocol        module.ProtocolInfo //2byte
	subProtocol     module.ProtocolInfo //2byte
	src             module.PeerID       //20byte
	dest            byte
	ttl             byte
	lengthOfpayload uint32 //4byte
	payload         []byte
	hashOfPacket    uint64 //8byte
	//Transient fields
}

const (
	p2pDestAny       = 0x00
	p2pDestPeerGroup = 0x08
	p2pDestPeer      = 0xFF
)

func NewPacket(subProtocol module.ProtocolInfo, payload []byte) *Packet {
	return &Packet{
		protocol:        PROTO_CONTOL,
		subProtocol:     subProtocol,
		lengthOfpayload: uint32(len(payload)),
		payload:         payload[:],
	}
}

func (p *Packet) String() string {
	return fmt.Sprintf("{pi:%#04x,subPi:%#04x,src:%v,dest:%#x,ttl:%d,len:%v,payload:[%X],hash:%#x}",
		p.protocol.Uint16(),
		p.subProtocol.Uint16(),
		p.src,
		p.dest,
		p.ttl,
		p.lengthOfpayload,
		p.payload,
		p.hashOfPacket)
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

func (pr *PacketReader) Reset() {
	pr.Reader.Reset(pr.rd)
}

func (pr *PacketReader) ReadPacket() (pkt *Packet, h hash.Hash64, e error) {
	for {
		if pr.pkt == nil {
			hb := make([]byte, packetHeaderSize)
			_, err := pr.Read(hb)
			if err != nil {
				e = err
				return
			}
			tb := hb[:]
			// pi := module.ProtocolInfo(binary.BigEndian.Uint16(tb[:2]))
			pi := NewProtocolInfo(tb[:2])
			tb = tb[2:]
			//spi := module.ProtocolInfo(binary.BigEndian.Uint16(tb[:2]))
			spi := NewProtocolInfo(tb[:2])
			tb = tb[2:]
			src := NewPeerID(tb[:peerIDSize])
			tb = tb[peerIDSize:]
			dest := tb[0]
			tb = tb[1:]
			ttl := tb[0]
			tb = tb[1:]
			lop := binary.BigEndian.Uint32(tb[:4])
			tb = tb[4:]
			pr.pkt = &Packet{protocol: pi, subProtocol: spi, src: src, dest: dest, ttl: ttl, lengthOfpayload: lop}
			h = fnv.New64a()
			h.Write(hb)
		}

		if pr.pkt.payload == nil {
			//TODO if pkt.lengthOfpayload > p.reader.Size()
			if pr.pkt.lengthOfpayload > uint32(pr.Buffered()) {
				continue
			}
			pr.pkt.payload = make([]byte, pr.pkt.lengthOfpayload)
			_, err := pr.Read(pr.pkt.payload)
			if err != nil {
				e = err
				return
			}
			h.Write(pr.pkt.payload)
		}

		if pr.pkt.hashOfPacket == 0 {
			if packetFooterSize > pr.Buffered() {
				continue
			}
			fb := make([]byte, packetFooterSize)
			_, err := pr.Read(fb)
			if err != nil {
				e = err
				return
			}
			tb := fb[:]
			pr.pkt.hashOfPacket = binary.BigEndian.Uint64(tb[:8])
			tb = tb[8:]

			pkt = pr.pkt
			pr.pkt = nil
			return
		}
	}

}

type PacketWriter struct {
	*bufio.Writer
	wr io.Writer
}

func NewPacketWriter(w io.Writer) *PacketWriter {
	return &PacketWriter{Writer: bufio.NewWriterSize(w, DefaultPacketBufferSize), wr: w}
}

func (pw *PacketWriter) Reset() {
	pw.Writer.Reset(pw.wr)
}

func (pw *PacketWriter) WritePacket(pkt *Packet) error {
	hb := make([]byte, packetHeaderSize)
	tb := hb[:]
	pkt.protocol.Copy(tb[:2])
	//binary.BigEndian.PutUint16(tb[:2], pkt.protocol.Uint16())
	tb = tb[2:]
	pkt.subProtocol.Copy(tb[:2])
	//binary.BigEndian.PutUint16(tb[:2], uint16(pkt.subProtocol))
	tb = tb[2:]
	pkt.src.Copy(tb[:peerIDSize])
	tb = tb[peerIDSize:]
	tb[0] = pkt.dest
	tb = tb[1:]
	tb[0] = pkt.ttl
	tb = tb[1:]
	binary.BigEndian.PutUint32(tb[:4], pkt.lengthOfpayload)
	tb = tb[4:]
	_, err := pw.Write(hb)
	if err != nil {
		log.Printf("PacketWriter.WritePacket hb %T %#v %s", err, err, err)
		return err
	}
	//
	payload := pkt.payload[:pkt.lengthOfpayload]
	_, err = pw.Write(payload)
	if err != nil {
		log.Printf("PacketWriter.WritePacket payload %T %#v %s", err, err, err)
		return err
	}
	//
	fb := make([]byte, packetFooterSize)
	tb = fb[:]
	if pkt.hashOfPacket == 0 {
		h := fnv.New64a()
		h.Write(hb)
		h.Write(payload)
		pkt.hashOfPacket = h.Sum64()
	}
	binary.BigEndian.PutUint64(tb[:8], pkt.hashOfPacket)
	tb = tb[8:]
	_, err = pw.Write(fb)
	if err != nil {
		log.Printf("PacketWriter.WritePacket fb %T %#v %s", err, err, err)
		return err
	}
	return nil
}

func (pw *PacketWriter) Write(b []byte) (int, error) {
	wn := 0
	for {
		n, err := pw.Writer.Write(b[wn:])
		wn += n
		if err != nil && err == io.ErrShortWrite {
			log.Println("PacketWriter.Write io.ErrShortWrite", err)
			time.Sleep(1 * time.Second)
			continue
		} else {
			return wn, err
		}
	}
}

func (pw *PacketWriter) Flush() error {
	for {
		err := pw.Writer.Flush()
		if err != nil && err == io.ErrShortWrite {
			log.Println("PacketWriter.Flush io.ErrShortWrite", err)
			time.Sleep(1 * time.Second)
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
		pkt, h, err := prw.rd.ReadPacket()
		if err != nil {
			return nil, err
		}
		if pkt.hashOfPacket != h.Sum64() {
			e := fmt.Sprintf("Invalid hashOfPacket:%x, expected:%x", pkt.hashOfPacket, h.Sum64())
			return pkt, errors.New(e)
		}
		prw.rpkt = pkt
	}
	return prw.rpkt, nil
}

func (prw *PacketReadWriter) Reset() {
	defer prw.mtx.Unlock()
	prw.mtx.Lock()
	prw.b.Reset()
	prw.rd.Reset()
	prw.wr.Reset()
	prw.rpkt = nil
	prw.wpkt = nil
}

type PacketPool struct {
	buckets     []map[uint64]*Packet
	len         []int
	cur         int
	numOfBucket int
	lenOfBucket int
	mtx         sync.RWMutex
}

func NewPacketPool(numOfBucket uint8, lenOfBucket uint16) *PacketPool {
	pp := &PacketPool{
		buckets:     make([]map[uint64]*Packet, numOfBucket),
		len:         make([]int, numOfBucket),
		cur:         0,
		numOfBucket: int(numOfBucket),
		lenOfBucket: int(lenOfBucket),
	}
	pp.buckets[0] = make(map[uint64]*Packet)
	return pp
}

func (pp *PacketPool) Put(pkt *Packet) {
	defer pp.mtx.Unlock()
	pp.mtx.Lock()

	m := pp.buckets[pp.cur]
	m[pkt.hashOfPacket] = pkt
	pp.len[pp.cur]++
	if pp.len[pp.cur] >= pp.lenOfBucket {
		pp.cur++
		if pp.cur >= pp.numOfBucket {
			pp.cur = 0
		}
		pp.buckets[pp.cur] = make(map[uint64]*Packet)
		pp.len[pp.cur] = 0
	}
}

func (pp *PacketPool) Clear() {
	defer pp.mtx.Unlock()
	pp.mtx.Lock()

	for i := 0; i < pp.numOfBucket; i++ {
		pp.buckets[i] = nil
	}
	pp.cur = 0
	pp.buckets[0] = make(map[uint64]*Packet)
}

func (pp *PacketPool) Contains(pkt *Packet) bool {
	defer pp.mtx.RUnlock()
	pp.mtx.RLock()

	cur := pp.cur
	for i := 0; i < pp.numOfBucket; i++ {
		m := pp.buckets[cur]
		if m == nil {
			return false
		}
		_, ok := m[pkt.hashOfPacket]
		if ok {
			return true
		}
		if cur < 1 {
			cur = pp.numOfBucket
		}
		cur--
	}
	return false
}
