package common

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log"
)

const (
	// AddressBytes indicate number of required bytes for address.
	AddressBytes = 20
)

type Address [AddressBytes + 1]byte

func (a *Address) IsContract() bool {
	return a[0] == 1
}

func (a Address) String() string {
	if a[0] == 1 {
		return "cx" + hex.EncodeToString(a[1:])
	} else {
		return "hx" + hex.EncodeToString(a[1:])
	}
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return a.SetString(s)
}

func (a *Address) SetString(s string) error {
	var isContract bool = false
	switch {
	case s[0:2] == "cx":
		isContract = true
		s = s[2:]
	case s[0:2] == "hx":
		s = s[2:]
	case s[0:2] == "0x":
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	if bytes, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		if err := a.SetBytes(isContract, bytes); err != nil {
			return err
		}
	}
	return nil
}

func (a *Address) SetBytes(ic bool, b []byte) error {
	if b == nil {
		return ErrorIllegalArgument
	}
	switch {
	case len(b) < AddressBytes:
		copy(a[AddressBytes-len(b)+1:], b)
	default:
		copy(a[1:], b)
	}
	if ic {
		a[0] = 1
	} else {
		a[0] = 0
	}
	return nil
}

func NewAccountAddress(b []byte) *Address {
	a := new(Address)
	a.SetBytes(false, b)
	return a
}

func NewContractAddress(b []byte) *Address {
	a := new(Address)
	a.SetBytes(true, b)
	return a
}

func NewAddressFromString(s string) *Address {
	a := new(Address)
	if err := a.SetString(s); err != nil {
		log.Panicln("FAIL to Address.SetString() for", s, err)
	}
	return a
}

func (a *Address) Equal(a2 *Address) bool {
	return bytes.Equal(a[:], a2[:])
}
