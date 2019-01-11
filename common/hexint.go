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
	if i == nil || i.Sign() == 0 {
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

func encodeHexNumber(neg bool, b []byte) string {
	s := hex.EncodeToString(b)
	if len(s) == 0 {
		return "0x0"
	}
	if s[0] == '0' {
		s = s[1:]
	}
	if neg {
		return "-0x" + s
	} else {
		return "0x" + s
	}
}

func decodeHexNumber(s string) (bool, []byte, error) {
	negative := false
	if len(s) > 0 && s[0] == '-' {
		negative = true
		s = s[1:]
	}
	if len(s) > 2 && s[0:2] == "0x" {
		s = s[2:]
	}
	if (len(s) % 2) == 1 {
		s = "0" + s
	}
	bs, err := hex.DecodeString(s)
	return negative, bs, err
}

func ParseInt(s string, bits int) (int64, error) {
	if v64, err := strconv.ParseInt(s, 0, bits); err == nil {
		return v64, nil
	}
	if negative, bs, err := decodeHexNumber(s); err == nil {
		if len(bs)*8 > bits {
			return 0, errors.New("OutOfRange")
		}
		u64 := BytesToUint64(bs)
		edge := (uint64(1)) << uint(bits-1)
		if negative {
			if u64 > edge {
				return 0, errors.New("OutOfRange")
			}
			return -int64(u64), nil
		} else {
			if u64 >= edge {
				return 0, errors.New("OutOfRange")
			}
			return int64(u64), nil
		}
	} else {
		return 0, err
	}
}

func ParseUint(s string, bits int) (uint64, error) {
	if v64, err := strconv.ParseUint(s, 0, bits); err == nil {
		return v64, nil
	}
	if negative, bs, err := decodeHexNumber(s); err == nil && !negative {
		if len(bs)*8 > bits {
			return 0, errors.New("OutOfRange")
		}
		return BytesToUint64(bs), nil
	} else {
		return 0, errors.New("IllegalFormat")
	}
}

func FormatInt(v int64) string {
	var bs []byte
	if v < 0 {
		bs = Uint64ToBytes(uint64(-v))
		return encodeHexNumber(true, bs)
	} else {
		bs = Uint64ToBytes(uint64(v))
		return encodeHexNumber(false, bs)
	}
}

func FormatUint(v uint64) string {
	return encodeHexNumber(false, Uint64ToBytes(v))
}

type HexInt struct {
	big.Int
}

func (i HexInt) String() string {
	return encodeHexNumber(i.Sign() < 0, i.Int.Bytes())
}

func (i HexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
		if _, ok := i.SetString(s, 0); ok {
			return nil
		}
		return err
	} else {
		neg, bs, err := decodeHexNumber(s)
		if err != nil {
			return err
		}
		i.Int.SetBytes(bs)
		if neg {
			i.Neg(&i.Int)
		}
		return nil
	}
}

func (i *HexInt) CodecEncodeSelf(e *codec.Encoder) {
	e.MustEncode(i.Bytes())
}

func (i *HexInt) CodecDecodeSelf(d *codec.Decoder) {
	var b []byte
	if err := d.Decode(&b); err == nil {
		i.SetBytes(b)
	}
}

func (i *HexInt) UnmarshalBinary(data []byte) error {
	i.SetBytes(data)
	return nil
}

func (i *HexInt) MarshalBinary() (data []byte, err error) {
	return i.Bytes(), nil
}

func (i *HexInt) Clone() HexInt {
	var v HexInt
	v.Set(&i.Int)
	return v
}

func (i *HexInt) Bytes() []byte {
	return BigIntToBytes(&i.Int)
}

func (i *HexInt) SetBytes(bs []byte) *big.Int {
	return BigIntSetBytes(&i.Int, bs)
}

func NewHexInt(v int64) *HexInt {
	i := new(HexInt)
	i.SetInt64(v)
	return i
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
	_ = e.Encode(i.Value)
}

func (i *HexInt16) CodecDecodeSelf(d *codec.Decoder) {
	_ = d.Decode(&i.Value)
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
	e.MustEncode(i.Value)
}

func (i *HexUint16) CodecDecodeSelf(d *codec.Decoder) {
	_ = d.Decode(&i.Value)
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
	_ = e.Encode(i.Value)
}

func (i *HexInt32) CodecDecodeSelf(d *codec.Decoder) {
	_ = d.Decode(&i.Value)
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
	_ = e.Encode(i.Value)
}

func (i *HexUint32) CodecDecodeSelf(e *codec.Decoder) {
	_ = e.Decode(&i.Value)
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
	_ = e.Encode(i.Value)
}

func (i *HexInt64) CodecDecodeSelf(e *codec.Decoder) {
	_ = e.Decode(&i.Value)
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
	_ = e.Encode(i.Value)
}

func (i *HexUint64) CodecDecodeSelf(e *codec.Decoder) {
	_ = e.Decode(&i.Value)
}
