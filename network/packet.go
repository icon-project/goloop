package network

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"io"

	"github.com/icon-project/goloop/module"
)

const (
	packetHeaderSize = 8 + peerIDSize
	packetHashSize   = 8
	packetFooterSize = packetHashSize
)

//srcPeerId, castType, destInfo, TTL(0:unlimited)
type Packet struct {
	protocol        module.ProtocolInfo //2byte
	subProtocol     module.ProtocolInfo //2byte
	src             module.PeerID       //20byte
	dest            byte                //1byte
	ttl             byte                //1byte
	lengthOfpayload uint32              //4byte
	payload         []byte
	hashOfPacket    []byte //8byte
}

func NewPacket(subProtocol module.ProtocolInfo, payload []byte) *Packet {
	return &Packet{
		subProtocol:     subProtocol,
		lengthOfpayload: uint32(len(payload)),
		payload:         payload[:],
	}
}

func (p *Packet) String() string {
	return fmt.Sprintf("{pi:%#04x,subPi:%#04x,src:%v,dest:%#x,ttl:%d,len:%v,payload:[%X],hash:%#x}",
		p.protocol,
		p.subProtocol,
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
			pi := module.ProtocolInfo(binary.BigEndian.Uint16(tb[:2]))
			tb = tb[2:]
			spi := module.ProtocolInfo(binary.BigEndian.Uint16(tb[:2]))
			tb = tb[2:]
			src := NewPeerId(tb[:peerIDSize])
			tb = tb[peerIDSize:]
			lop := binary.BigEndian.Uint32(tb[:4])
			tb = tb[4:]
			pr.pkt = &Packet{protocol: pi, subProtocol: spi, src: src, lengthOfpayload: lop}
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

		if pr.pkt.hashOfPacket == nil {
			if packetFooterSize > pr.Buffered() {
				continue
			}
			pr.pkt.hashOfPacket = make([]byte, packetHashSize)
			_, err := pr.Read(pr.pkt.hashOfPacket)
			if err != nil {
				e = err
				return
			}
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
	binary.BigEndian.PutUint16(tb[:2], uint16(pkt.protocol))
	tb = tb[2:]
	binary.BigEndian.PutUint16(tb[:2], uint16(pkt.subProtocol))
	tb = tb[2:]
	pkt.src.Copy(tb[:peerIDSize])
	tb = tb[peerIDSize:]
	binary.BigEndian.PutUint32(tb[:4], pkt.lengthOfpayload)
	tb = tb[4:]
	_, err := pw.Write(hb)
	if err != nil {
		return err
	}
	//
	payload := pkt.payload[:pkt.lengthOfpayload]
	_, err = pw.Write(payload)
	if err != nil {
		return err
	}
	//
	if pkt.hashOfPacket == nil {
		h := fnv.New64a()
		h.Write(hb)
		h.Write(payload)
		pkt.hashOfPacket = make([]byte, packetHashSize)
		binary.BigEndian.PutUint64(pkt.hashOfPacket, h.Sum64())
	}
	_, err = pw.Write(pkt.hashOfPacket)
	return err
}

type PacketReadWriter struct {
	*PacketReader
	*PacketWriter
}

func NewPacketReadWriter() *PacketReadWriter {
	buf := bytes.NewBuffer(make([]byte, DefaultPacketBufferSize))
	return &PacketReadWriter{NewPacketReader(buf), NewPacketWriter(buf)}
}
