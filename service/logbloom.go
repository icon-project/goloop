package service

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/ugorji/go/codec"
	"log"
	"math/big"
	"regexp"
	"strings"
)

const (
	LogBloomBits  = 2048
	LogBloomBytes = LogBloomBits / 8
	LogBloomWords = LogBloomBytes / 4
)

const (
	configLogBloomLegacy = true
)

/*
	Sample event data
	{
		"scoreAddress":"cx88ff9111d2361d380030e9d79bbf8b11671f1ada",
    	"indexed": [EventAccountRegistered(Address,int,int), hxca916987102102dcee50e5109346b6ee767bc2bd],
		"data": [0x3635c9adc5dea00000, 0x43c33c1937564800000]
	}
*/

// logBloom store blooms of logs.
type logBloom struct {
	big.Int
}

func (lb *logBloom) String() string {
	return "0x" + hex.EncodeToString(lb.LogBytes())
}
func (lb *logBloom) LogBytes() []byte {
	bs := make([]byte, LogBloomBytes)
	ibs := lb.Int.Bytes()
	copy(bs[LogBloomBytes-len(ibs):], ibs)
	return bs
}

func (lb logBloom) MarshalJSON() ([]byte, error) {
	s := "0x" + hex.EncodeToString(lb.LogBytes())
	return json.Marshal(s)
}

func (lb *logBloom) UnmarshalJSON(data []byte) error {
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

func (lb *logBloom) CodecEncodeSelf(e *codec.Encoder) {
	b := lb.Bytes()
	e.Encode(b)
}

func (lb *logBloom) CodecDecodeSelf(d *codec.Decoder) {
	var b []byte
	d.Decode(&b)
	lb.SetBytes(b)
}

// Merge bloom
func (lb *logBloom) Merge(lb2 *logBloom) {
	lb.Int.Or(&lb.Int, &lb2.Int)
}

// Contain checks whether it includes the bloom
func (lb *logBloom) Contain(lb2 *logBloom) bool {
	var r big.Int
	r.And(&lb.Int, &lb2.Int)
	return r.Cmp(&lb2.Int) == 0
}

func (lb *logBloom) addBit(idx uint16) {
	lb.Int.SetBit(&lb.Int, int(idx), 1)
}

func decomposeSignature(s string) (string, []string) {
	reg := regexp.MustCompile(`^(\w+)\(((?:\w+)(?:,(?:\w+))*)\)$`)
	if reg == nil {
		return "", nil
	}
	matches := reg.FindStringSubmatch(s)
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], strings.Split(matches[2], ",")
}

func (lb *logBloom) AddEvent(e *eventLog) error {
	if len(e.Indexed) < 1 {
		return nil
	}
	ws := make([]string, 0, len(e.Indexed))
	ws = append(ws, string([]byte{0})+e.Indexed[0])
	_, pts := decomposeSignature(e.Indexed[0])
	for i, is := range e.Indexed[1:] {
		if i >= len(pts) {
			break
		}
		w := []byte{byte(i + 1)}
		switch pts[i] {
		case "Address":
			var addr common.Address
			if err := addr.SetString(is); err != nil {
				return err
			}
			w = append(w, addr.ID()...)
		case "int":
			var value big.Int
			value.SetString(is, 0)
			w = append(w, common.BigIntToBytes(&value)...)
		case "str":
			w = append(w, []byte(is)...)
		case "bytes":
			bs, err := hex.DecodeString(is[2:])
			if err != nil {
				return err
			}
			w = append(w, bs...)
		case "bool":
			if is == "0x1" {
				w = append(w, 1)
			} else {
				w = append(w, 0)
			}
		default:
			log.Panicf("Unknown parameter type=%s", pts[i])
		}
		ws = append(ws, string(w))
	}
	lb.AddLog(ws...)
	return nil
}

func (lb *logBloom) AddLog(logs ...string) {
	for _, log := range logs {
		var h []byte
		if configLogBloomLegacy {
			h = crypto.SHASum256([]byte(log))
			h = []byte(hex.EncodeToString(h))
		} else {
			h = crypto.SHA3Sum256([]byte(log))
		}
		for i := 0; i < 3; i++ {
			lb.addBit(binary.BigEndian.Uint16(h[i*2:i*2+2]) & (LogBloomBits - 1))
		}
	}
}
