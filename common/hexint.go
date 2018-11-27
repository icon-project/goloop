package common

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
	"github.com/ugorji/go/codec"
)

var BigIntOne = big.NewInt(1)

func BigIntToBytes(i *big.Int) []byte {
	if i.Sign() == 0 {
		return []byte{}
	} else if i.Sign() > 0 {
		bl := i.BitLen()
		if (bl % 8) == 0 {
			bs := make([]byte, bl/8+1)
			copy(bs[1:], i.Bytes())
			return bs
		}
		return i.Bytes()
	} else {
		var ti, nb big.Int
		ti.Add(i, BigIntOne)
		bl := ti.BitLen()
		nb.SetBit(&nb, (bl+8)/8*8, 1)
		nb.Add(&nb, i)
		return nb.Bytes()
	}
}

func BigIntSetBytes(i *big.Int, bs []byte) *big.Int {
	i.SetBytes(bs)
	if len(bs) > 0 && (bs[0]&0x80) != 0 {
		var base big.Int
		base.SetBit(&base, i.BitLen(), 1)
		i.Sub(i, &base)
	}
	return i
}

func Uint64ToBytes(v uint64) []byte {
	if v == 0 {
		return []byte{}
	}
	bs := make([]byte, 8)
	for idx := 7; idx >= 0; idx-- {
		bs[idx] = byte(v & 0xff)
		v >>= 8
		if v == 0 {
			return bs[idx:]
		}
	}
	return bs
}

func BytesToUint64(bs []byte) uint64 {
	var v uint64
	for _, b := range bs {
		v = (v << 8) | uint64(b)
	}
	return v
}

func BytesToInt64(bs []byte) int64 {
	if len(bs) == 0 {
		return 0
	}
	var v int64
	if (bs[0] & 0x80) != 0 {
		for _, b := range bs {
			v = (v << 8) | int64(b^0xff)
		}
		return -v - 1
	} else {
		for _, b := range bs {
			v = (v << 8) | int64(b)
		}
		return v
	}
}

func Int64ToBytes(v int64) []byte {
	if v == 0 {
		return []byte{}
	}
	bs := make([]byte, 8)

	const mask int64 = -0x80
	var target int64 = 0
	if v < 0 {
		target = mask
	}
	for idx := 7; idx >= 0; idx-- {
		bs[idx] = byte(v & 0xff)
		if (v & mask) == target {
			return bs[idx:]
		}
		v >>= 8
	}
	return bs
}

func encodeHexNumber(b []byte) string {
	s := hex.EncodeToString(b)
	if len(s) == 0 {
		return "0x0"
	}
	if s[0] == '0' {
		return "0x" + s[1:]
	}
	return "0x" + s
}

func decodeHexNumber(s string) ([]byte, error) {
	if len(s) > 2 && s[0:2] == "0x" {
		s = s[2:]
	}
	if (len(s) % 2) == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

func ParseInt(s string, bits int) (int64, error) {
	if bs, err := decodeHexNumber(s); err == nil {
		if len(bs)*8 > bits {
			return 0, errors.New("OutOfRange")
		}
		return BytesToInt64(bs), nil
	}
	return strconv.ParseInt(s, 0, bits)
}

func ParseUint(s string, bits int) (uint64, error) {
	if bs, err := decodeHexNumber(s); err == nil {
		if len(bs)*8 > bits {
			return 0, errors.New("OutOfRange")
		}
		return BytesToUint64(bs), nil
	}
	return strconv.ParseUint(s, 0, bits)
}

func FormatInt(v int64) string {
	return encodeHexNumber(Int64ToBytes(v))
}

func FormatUint(v uint64) string {
	return encodeHexNumber(Uint64ToBytes(v))
}

type HexUint struct {
	big.Int
}

func (i HexUint) String() string {
	return "0x" + i.Text(16)
}

func (i HexUint) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
		if _, ok := i.SetString(s, 0); ok {
			return nil
		}
		return err
	} else {
		if len(s) >= 2 && s[0:2] == "0x" {
			s = s[2:]
		}
		if _, ok := i.SetString(s, 16); ok {
			return nil
		}
	}
	return errors.Errorf("FailToParse(%s) for HexUint", s)
}

func (i *HexUint) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Bytes())
}

func (i *HexUint) CodecDecodeSelf(d *codec.Decoder) {
	var b []byte
	if err := d.Decode(&b); err == nil {
		i.SetBytes(b)
	}
}

func (i *HexUint) Clone() HexUint {
	var v HexUint
	v.Set(&i.Int)
	return v
}

type HexInt struct {
	big.Int
}

func (i HexInt) String() string {
	s := hex.EncodeToString(i.Bytes())
	if len(s) == 0 {
		return "0x0"
	}
	if s[0] == '0' {
		return "0x" + s[1:]
	} else {
		return "0x" + s
	}
}

func (i *HexInt) SetString(s string) (*big.Int, bool) {
	if len(s) > 2 && s[0:2] == "0x" {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		return i.Int.SetString(s, 16)
	}
	if bs, err := hex.DecodeString(s); err != nil {
		return nil, false
	} else {
		return i.SetBytes(bs), true
	}
}

func (i *HexInt) Bytes() []byte {
	return BigIntToBytes(&i.Int)
}

func (i *HexInt) SetBytes(b []byte) *big.Int {
	return BigIntSetBytes(&i.Int, b)
}

func (i HexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
		if _, ok := i.Int.SetString(s, 0); ok {
			return nil
		}
		return err
	} else {
		if _, ok := i.SetString(s); ok {
			return nil
		}
	}
	return errors.Errorf("FailToParse(%s) for HexInt", s)
}

func (i *HexInt) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Bytes())
}

func (i *HexInt) CodecDecodeSelf(d *codec.Decoder) {
	var b []byte
	if err := d.Decode(&b); err == nil {
		i.SetBytes(b)
	}
}

func (i *HexInt) Clone() HexInt {
	var v HexInt
	v.Set(&i.Int)
	return v
}

type HexInt16 struct {
	Value int16
}

func (i HexInt16) String() string {
	return FormatInt(int64(i.Value))
}

func (i HexInt16) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt16) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := ParseInt(s, 16); err == nil {
		i.Value = int16(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexInt16) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Value)
}

func (i *HexInt16) CodecDecodeSelf(e *codec.Decoder) {
	e.Decode(&i.Value)
}

func (i HexInt16) Bytes() []byte {
	return Int64ToBytes(int64(i.Value))
}

type HexUint16 struct {
	Value uint16
}

func (i HexUint16) String() string {
	return FormatInt(int64(i.Value))
}

func (i HexUint16) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint16) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := ParseUint(s, 16); err == nil {
		i.Value = uint16(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexUint16) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Value)
}

func (i *HexUint16) CodecDecodeSelf(e *codec.Decoder) {
	e.Decode(&i.Value)
}

func (i HexUint16) Bytes() []byte {
	return Uint64ToBytes(uint64(i.Value))
}

type HexInt32 struct {
	Value int32
}

func (i HexInt32) String() string {
	return FormatInt(int64(i.Value))
}

func (i HexInt32) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt32) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := ParseInt(s, 32); err == nil {
		i.Value = int32(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexInt32) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Value)
}

func (i *HexInt32) CodecDecodeSelf(e *codec.Decoder) {
	e.Decode(&i.Value)
}

type HexUint32 struct {
	Value uint32
}

func (i HexUint32) String() string {
	return FormatUint(uint64(i.Value))
}

func (i HexUint32) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint32) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := ParseUint(s, 32); err == nil {
		i.Value = uint32(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexUint32) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Value)
}

func (i *HexUint32) CodecDecodeSelf(e *codec.Decoder) {
	e.Decode(&i.Value)
}

type HexInt64 struct {
	Value int64
}

func (i HexInt64) String() string {
	return FormatInt(i.Value)
}

func (i HexInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := ParseInt(s, 64); err == nil {
		i.Value = v64
		return nil
	} else {
		return err
	}
}

func (i *HexInt64) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Value)
}

func (i *HexInt64) CodecDecodeSelf(e *codec.Decoder) {
	e.Decode(&i.Value)
}

type HexUint64 struct {
	Value uint64
}

func (i HexUint64) String() string {
	return FormatUint(i.Value)
}

func (i HexUint64) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := ParseUint(s, 64); err == nil {
		i.Value = v64
		return nil
	} else {
		return err
	}
}

func (i *HexUint64) CodecEncodeSelf(e *codec.Encoder) {
	e.Encode(i.Value)
}

func (i *HexUint64) CodecDecodeSelf(e *codec.Decoder) {
	e.Decode(&i.Value)
}
