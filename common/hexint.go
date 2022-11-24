package common

import (
	"encoding/json"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/intconv"
)

var hexIntZero HexInt
var HexIntZero = &hexIntZero

type HexInt struct {
	big.Int
}

func (i HexInt) String() string {
	return intconv.FormatBigInt(&i.Int)
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
		return intconv.ParseBigInt(&i.Int, s)
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
	return intconv.BigIntToBytes(&i.Int)
}

func (i *HexInt) SetBytes(bs []byte) *big.Int {
	return intconv.BigIntSetBytes(&i.Int, bs)
}

func (i *HexInt) Value() *big.Int {
	if i == nil {
		return nil
	}
	return &i.Int
}

func (i *HexInt) SetValue(x *big.Int) *HexInt {
	if iv := i.Int.Set(x); iv == nil {
		return nil
	} else {
		return i
	}
}

func (i *HexInt) AddValue(x, y *big.Int) *HexInt {
	i.Add(x, y)
	return i
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
	return intconv.FormatInt(int64(i.Value))
}

func (i HexInt16) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt16) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := intconv.ParseInt(s, 16); err == nil {
		i.Value = int16(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexInt16) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(int64(i.Value))
}

func (i *HexInt16) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}

func (i HexInt16) Bytes() []byte {
	return intconv.Int64ToBytes(int64(i.Value))
}

type HexUint16 struct {
	Value uint16
}

func (i HexUint16) String() string {
	return intconv.FormatUint(uint64(i.Value))
}

func (i HexUint16) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint16) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := intconv.ParseUint(s, 16); err == nil {
		i.Value = uint16(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexUint16) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(int64(i.Value))
}

func (i *HexUint16) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}

func (i HexUint16) Bytes() []byte {
	return intconv.Int64ToBytes(int64(i.Value))
}

type HexInt32 struct {
	Value int32
}

func (i HexInt32) String() string {
	return intconv.FormatInt(int64(i.Value))
}

func (i HexInt32) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt32) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := intconv.ParseInt(s, 32); err == nil {
		i.Value = int32(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexInt32) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(int64(i.Value))
}

func (i *HexInt32) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}

type HexUint32 struct {
	Value uint32
}

func (i HexUint32) String() string {
	return intconv.FormatUint(uint64(i.Value))
}

func (i HexUint32) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint32) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := intconv.ParseUint(s, 32); err == nil {
		i.Value = uint32(v64)
		return nil
	} else {
		return err
	}
}

func (i *HexUint32) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(int64(i.Value))
}

func (i *HexUint32) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}

type HexInt64 struct {
	Value int64
}

func (i HexInt64) String() string {
	return intconv.FormatInt(i.Value)
}

func (i HexInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexInt64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := intconv.ParseInt(s, 64); err == nil {
		i.Value = v64
		return nil
	} else {
		return err
	}
}

func (i *HexInt64) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(int64(i.Value))
}

func (i *HexInt64) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}

type HexUint64 struct {
	Value uint64
}

func (i HexUint64) String() string {
	return intconv.FormatUint(i.Value)
}

func (i HexUint64) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func (i *HexUint64) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		s = string(b)
	}
	if v64, err := intconv.ParseUint(s, 64); err == nil {
		i.Value = v64
		return nil
	} else {
		return err
	}
}

func (i *HexUint64) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(i.Value)
}

func (i *HexUint64) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&i.Value)
}
