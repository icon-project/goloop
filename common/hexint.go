package common

import (
	"encoding/json"
	"github.com/ugorji/go/codec"
	"math/big"
	"strconv"
)

type HexInt struct {
	big.Int
}

func (i HexInt) String() string {
	return "0x" + i.Text(16)
}

func (i *HexInt) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	_, ok := i.SetString(s, 0)
	if ok {
		return nil
	}
	return ErrIllegalArgument
}

func (i *HexInt) CodecEncodeSelf(e *codec.Encoder) {
	b := i.Int.Bytes()
	e.Encode(b)
}

func (i *HexInt) CodecDecodeSelf(d *codec.Decoder) {
	var b []byte
	if err := d.Decode(&b); err == nil {
		i.Int.SetBytes(b)
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
	return "0x" + strconv.FormatInt(int64(i.Value), 16)
}

func (i *HexInt16) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := strconv.ParseInt(s, 0, 16); err == nil {
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

type HexUint16 struct {
	Value uint16
}

func (i HexUint16) String() string {
	return "0x" + strconv.FormatUint(uint64(i.Value), 16)
}

func (i *HexUint16) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := strconv.ParseUint(s, 0, 16); err == nil {
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

type HexInt32 struct {
	Value int32
}

func (i HexInt32) String() string {
	return "0x" + strconv.FormatInt(int64(i.Value), 16)
}

func (i *HexInt32) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := strconv.ParseInt(s, 0, 32); err == nil {
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
	return "0x" + strconv.FormatUint(uint64(i.Value), 16)
}

func (i *HexUint32) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := strconv.ParseUint(s, 0, 32); err == nil {
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
	return "0x" + strconv.FormatInt(i.Value, 16)
}

func (i *HexInt64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := strconv.ParseInt(s, 0, 64); err == nil {
		i.Value = int64(v64)
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
	return "0x" + strconv.FormatUint(uint64(i.Value), 16)
}

func (i *HexUint64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := strconv.ParseUint(s, 0, 64); err == nil {
		i.Value = uint64(v64)
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
