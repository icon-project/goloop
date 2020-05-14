package txresult

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	LogsBloomBits  = 2048
	LogsBloomBytes = LogsBloomBits / 8
)

const (
	configLogsBloomSHA256      = false
	configLogsBloomIncludeAddr = true
)

// logsBloom store blooms of logs.
type LogsBloom struct {
	big.Int
}

func NewLogsBloom(bs []byte) *LogsBloom {
	lb := &LogsBloom{}
	lb.SetBytes(bs)
	return lb
}

func (lb *LogsBloom) String() string {
	return "0x" + hex.EncodeToString(lb.LogBytes())
}
func (lb *LogsBloom) LogBytes() []byte {
	bs := make([]byte, LogsBloomBytes)
	ibs := lb.Int.Bytes()
	copy(bs[LogsBloomBytes-len(ibs):], ibs)
	return bs
}

func (lb LogsBloom) MarshalJSON() ([]byte, error) {
	s := "0x" + hex.EncodeToString(lb.LogBytes())
	return json.Marshal(s)
}

func (lb *LogsBloom) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if _, ok := lb.SetString(s, 0); !ok {
		lb.SetInt64(0)
		return errors.ErrIllegalArgument
	}
	return nil
}

func (lb *LogsBloom) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(lb.Bytes())
}

func (lb *LogsBloom) RLPDecodeSelf(d codec.Decoder) error {
	var bs []byte
	if err := d.Decode(&bs); err != nil {
		return err
	}
	lb.SetBytes(bs)
	return nil
}

// Merge bloom
func (lb *LogsBloom) Merge(lb2 module.LogsBloom) {
	if lb2 == nil {
		return
	}
	var lb2Ptr *LogsBloom
	if ptr, ok := lb2.(*LogsBloom); ok {
		lb2Ptr = ptr
	} else {
		lb2Ptr = new(LogsBloom)
		lb2Ptr.SetBytes(lb2.Bytes())
	}
	lb.Int.Or(&lb.Int, &lb2Ptr.Int)
}

// Contain checks whether it includes the bloom
func (lb *LogsBloom) Contain(mlb module.LogsBloom) bool {
	lb2, ok := mlb.(*LogsBloom)
	if !ok {
		lbs := mlb.Bytes()
		lb2 = new(LogsBloom)
		lb2.SetBytes(lbs)
	}
	words1 := lb.Bits()
	words2 := lb2.Bits()
	if len(words2) > len(words1) {
		return false
	}
	for idx, word2 := range words2 {
		if word2 == 0 {
			continue
		}
		word1 := words1[idx]
		if (word1 & word2) != word2 {
			return false
		}
	}
	return true
}

func (lb *LogsBloom) Equal(mlb module.LogsBloom) bool {
	return bytes.Equal(lb.Bytes(), mlb.Bytes())
}

func (lb *LogsBloom) addBit(idx uint16) {
	lb.Int.SetBit(&lb.Int, int(idx), 1)
}

func (lb *LogsBloom) AddLog(addr module.Address, log [][]byte) {
	if len(log) == 0 {
		return
	}
	if configLogsBloomIncludeAddr {
		lb.AddAddressOfLog(addr)
	}
	for i, b := range log {
		if b == nil {
			continue
		}
		lb.AddIndexedOfLog(i, b)
	}
}

func (lb *LogsBloom) AddAddressOfLog(addr module.Address) {
	bs := make([]byte, common.AddressBytes+1)
	bs[0] = 0xff
	copy(bs[1:], addr.Bytes())
	lb.addLog(bs)
}

func (lb *LogsBloom) AddIndexedOfLog(i int, b []byte) {
	bs := make([]byte, len(b)+1)
	bs[0] = byte(i)
	copy(bs[1:], b)
	lb.addLog(bs)
}

func (lb *LogsBloom) addLog(log []byte) {
	var h []byte
	if configLogsBloomSHA256 {
		h = crypto.SHASum256(log)
		h = []byte(hex.EncodeToString(h))
	} else {
		h = crypto.SHA3Sum256(log)
	}
	for i := 0; i < 3; i++ {
		lb.addBit(binary.BigEndian.Uint16(h[i*2:i*2+2]) & (LogsBloomBits - 1))
	}
}

func NewLogsBloomFromCompressed(bs []byte) *LogsBloom {
	lb := &LogsBloom{}
	lb.SetCompressedBytes(bs)

	return lb
}

func (lb *LogsBloom) CompressedBytes() []byte {
	return common.Compress(lb.Bytes())
}

func (lb *LogsBloom) SetCompressedBytes(bs []byte) *big.Int {
	return lb.SetBytes(common.Decompress(bs))
}
