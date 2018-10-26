package common

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strconv"
)

func DecodeHexNumber(s string) ([]byte, error) {
	var i big.Int
	_, ok := i.SetString(s, 0)
	if ok {
		return i.Bytes(), nil
	} else {
		return nil, ErrIllegalArgument
	}
}

func FixBytes(b []byte, l int) []byte {
	bl := len(b)
	switch {
	case bl == l:
		return b
	case bl > l:
		return b[bl-l:]
	}
	return append(bytes.Repeat([]byte{0}, l-bl), b...)
}

type HexInt struct {
	big.Int
}

func (i HexInt) String() string {
	return "0x" + i.Text(16)
}

func (i *HexInt) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	_, ok := i.SetString(s, 0)
	if ok {
		return nil
	}
	return ErrIllegalArgument
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
