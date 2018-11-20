package service

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/ugorji/go/codec"
)

const (
	LogBloomBits  = 2048
	LogBloomBytes = LogBloomBits / 8
	LogBloomWords = LogBloomBytes / 4
)

/*
	Sample data
	{
		"scoreAddress":"cx88ff9111d2361d380030e9d79bbf8b11671f1ada",
    	"indexed": [EventAccountRegistered(Address,int,int), hxca916987102102dcee50e5109346b6ee767bc2bd],
		"data": [0x3635c9adc5dea00000, 0x43c33c1937564800000]
	}
*/

// LogBloom store blooms of logs.
type LogBloom struct {
	data [LogBloomWords]uint32
}

func (lb *LogBloom) Bytes() []byte {
	b := make([]byte, LogBloomBytes)
	for i, u32 := range lb.data {
		binary.BigEndian.PutUint32(b[i*4:i*4+4], u32)
	}
	return b
}

func (lb *LogBloom) SetBytes(b []byte) {
	for i := 0; i+4 <= len(b); i += 4 {
		lb.data[i/4] = binary.BigEndian.Uint32(b[i : i+4])
	}
}

func (lb *LogBloom) String() string {
	return "0x" + hex.EncodeToString(lb.Bytes())
}

func (lb *LogBloom) CodecEncodeSelf(e *codec.Encoder) {
	b := lb.Bytes()
	e.Encode(b)
}

func (lb *LogBloom) CodecDecodeSelf(d *codec.Decoder) {
	var b []byte
	d.Decode(&b)
	lb.SetBytes(b)
}

// Merge bloom
func (lb *LogBloom) Merge(lb2 *LogBloom) {
	for i, u32 := range lb2.data {
		lb.data[i] |= u32
	}
}

// Contain checks whether it includes the bloom
func (lb *LogBloom) Contain(lb2 *LogBloom) bool {
	for i, u32 := range lb2.data {
		if (lb.data[i] & u32) != u32 {
			return false
		}
	}
	return true
}

func (lb *LogBloom) addBit(idx uint16) {
	lb.data[idx/32] |= uint32(1) << (31 - (idx % 32))
}

func (lb *LogBloom) AddLog(logs ...string) {
	for _, log := range logs {
		h := crypto.SHA3Sum256([]byte(log))
		for i := 0; i < 3; i++ {
			lb.addBit(binary.BigEndian.Uint16(h[i*2:i*2+2]) & 2047)
		}
	}
}
