package common

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/pkg/errors"
	"github.com/ugorji/go/codec"
	"math/big"
)

const (
	LogBloomBits  = 2048
	LogBloomBytes = LogBloomBits / 8
)

const (
	configLogBloomLegacy = true
)

// logBloom store blooms of logs.
type LogBloom struct {
	big.Int
}

func (lb *LogBloom) String() string {
	return "0x" + hex.EncodeToString(lb.LogBytes())
}
func (lb *LogBloom) LogBytes() []byte {
	bs := make([]byte, LogBloomBytes)
	ibs := lb.Int.Bytes()
	copy(bs[LogBloomBytes-len(ibs):], ibs)
	return bs
}

func (lb LogBloom) MarshalJSON() ([]byte, error) {
	s := "0x" + hex.EncodeToString(lb.LogBytes())
	return json.Marshal(s)
}

func (lb *LogBloom) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if _, ok := lb.SetString(s, 0); !ok {
		lb.SetInt64(0)
		return errors.New("IllegalArgument")
	}
	return nil
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
	lb.Int.Or(&lb.Int, &lb2.Int)
}

// Contain checks whether it includes the bloom
func (lb *LogBloom) Contain(lb2 *LogBloom) bool {
	var r big.Int
	r.And(&lb.Int, &lb2.Int)
	return r.Cmp(&lb2.Int) == 0
}

func (lb *LogBloom) addBit(idx uint16) {
	lb.Int.SetBit(&lb.Int, int(idx), 1)
}

func (lb *LogBloom) AddLog(log [][]byte) {
	if len(log) == 0 {
		return
	}
	for i, b := range log {
		bs := make([]byte, len(b)+1)
		bs[0] = byte(i)
		copy(bs[1:], b)
		lb.addLog(bs)
	}
}

func (lb *LogBloom) addLog(logs ...[]byte) {
	for _, log := range logs {
		var h []byte
		if configLogBloomLegacy {
			h = crypto.SHASum256(log)
			h = []byte(hex.EncodeToString(h))
		} else {
			h = crypto.SHA3Sum256(log)
		}
		for i := 0; i < 3; i++ {
			lb.addBit(binary.BigEndian.Uint16(h[i*2:i*2+2]) & (LogBloomBits - 1))
		}
	}
}
