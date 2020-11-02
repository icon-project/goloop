package common

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	AddressIDBytes = 20
	AddressBytes   = AddressIDBytes + 1
)

type Address [AddressBytes]byte

func (a *Address) IsContract() bool {
	return a[0] == 1
}

func (a *Address) String() string {
	if a[0] == 1 {
		return "cx" + hex.EncodeToString(a[1:])
	} else {
		return "hx" + hex.EncodeToString(a[1:])
	}
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	return a.SetString(s)
}

func (a *Address) SetString(s string) error {
	var isContract = false
	if len(s) >= 2 {
		switch {
		case s[0:2] == "cx":
			isContract = true
			s = s[2:]
		case s[0:2] == "hx":
			s = s[2:]
		case s[0:2] == "0x":
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	if bytes, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		a.SetTypeAndID(isContract, bytes)
	}
	return nil
}

func (a *Address) Bytes() []byte {
	return (*a)[:]
}

// BytesPart returns part of address without type prefix.
func (a *Address) ID() []byte {
	return (*a)[1:]
}

func (a *Address) SetBytes(b []byte) error {
	if len(b) != AddressBytes {
		return ErrIllegalArgument
	}
	switch b[0] {
	case 0, 1:
		copy(a[:], b)
		return nil
	default:
		return ErrIllegalArgument
	}
}

var zeroBuffer [AddressIDBytes]byte

func (a *Address) SetTypeAndID(ic bool, id []byte) {
	switch {
	case len(id) < AddressIDBytes:
		bp := 1 + AddressIDBytes - len(id)
		copy(a[1:bp], zeroBuffer[:])
		copy(a[bp:], id)
	default:
		copy(a[1:], id)
	}
	if ic {
		a[0] = 1
	} else {
		a[0] = 0
	}
}

func NewAccountAddress(b []byte) *Address {
	a := new(Address)
	a.SetTypeAndID(false, b)
	return a
}

func NewAddress(b []byte) (*Address, error) {
	if len(b) == 0 {
		return nil, ErrIllegalArgument
	}
	a := new(Address)
	if err := a.SetBytes(b); err != nil {
		return nil, err
	}
	return a, nil
}

func NewAddressWithTypeAndID(isContract bool, id []byte) *Address {
	a := new(Address)
	a.SetTypeAndID(isContract, id)
	return a
}

func MustNewAddress(b []byte) *Address {
	if addr, err := NewAddress(b); err == nil {
		return addr
	} else {
		panic(err)
	}
}

func BytesToAddress(b []byte) (module.Address, error) {
	if len(b) == 0 {
		return nil, nil
	}
	return NewAddress(b)
}

func (a *Address) Set(address module.Address) *Address {
	if address != nil {
		a.SetTypeAndID(address.IsContract(), address.ID())
	}
	return a
}

func AddressToPtr(addr module.Address) *Address {
	if addr == nil {
		return nil
	}
	if addrPtr, ok := addr.(*Address); ok {
		return addrPtr
	} else {
		addrPtr = new(Address)
		addrPtr.SetTypeAndID(addr.IsContract(), addr.ID())
		return addrPtr
	}
}

func NewContractAddress(b []byte) *Address {
	a := new(Address)
	a.SetTypeAndID(true, b)
	return a
}

func NewAddressFromString(s string) *Address {
	a := new(Address)
	if err := a.SetString(s); err != nil {
		log.Panicln("FAIL to Address.SetString() for", s, err)
	}
	return a
}

func NewAccountAddressFromPublicKey(pubKey *crypto.PublicKey) *Address {
	pk := pubKey.SerializeUncompressed()
	if pk == nil {
		log.Panicln("FAIL invalid public key:", pubKey)
	}
	digest := crypto.SHA3Sum256(pk[1:])
	return NewAccountAddress(digest[len(digest)-AddressIDBytes:])
}

func (a *Address) Equal(a2 module.Address) bool {
	a2IsNil := a2 == nil || reflect.ValueOf(a2).IsNil()
	if a2IsNil && a == nil {
		return true
	}
	if a2IsNil || a == nil {
		return false
	}
	return bytes.Equal(a[:], a2.Bytes())
}

func (a *Address) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode([]byte(a[:]))
}

func (a *Address) RLPDecodeSelf(d codec.Decoder) error {
	if bs, err := d.DecodeBytes(); err != nil {
		return err
	} else {
		return a.SetBytes(bs)
	}
}
