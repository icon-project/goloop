package common

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"

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

func (a *Address) SetStringStrict(s string) error {
	if len(s) != AddressIDBytes*2+2 {
		return ErrIllegalArgument
	}
	prefix := s[0:2]
	body := s[2:]
	var isContract bool
	switch prefix {
	case "cx":
		isContract = true
	case "hx":
	default:
		return ErrIllegalArgument
	}
	if strings.ToLower(body) != body {
		return ErrIllegalArgument
	}
	if bytes, err := hex.DecodeString(body); err != nil {
		return err
	} else {
		a.SetTypeAndID(isContract, bytes)
		return nil
	}
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
	if blen := len(b); blen == AddressBytes {
		switch b[0] {
		case 0, 1:
			copy(a[:], b)
			return nil
		default:
			return ErrIllegalArgument
		}
	} else if blen == AddressIDBytes {
		a[0] = 0
		copy(a[1:], b)
		return nil
	} else {
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
	return NewAddressWithTypeAndID(false, b)
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
		return NewAddressWithTypeAndID(addr.IsContract(), addr.ID())
	}
}

func NewContractAddress(b []byte) *Address {
	return NewAddressWithTypeAndID(true, b)
}

func MustNewAddressFromString(s string) *Address {
	if addr, err := NewAddressFromString(s); err != nil {
		log.Panicf("FAIL to create address with string=%q", s)
		return nil
	} else {
		return addr
	}
}

func NewAddressFromString(s string) (*Address, error) {
	a := new(Address)
	if err := a.SetString(s); err != nil {
		return nil, err
	} else {
		return a, nil
	}
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
	return a.IsContract() == a2.IsContract() && bytes.Equal(a.ID(), a2.ID())
}

func AddressEqual(a, b module.Address) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(b)
}

func BytesOfAddress(addr module.Address) []byte {
	if addr == nil {
		return nil
	}
	return addr.Bytes()
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

func ToAddress(addr interface{}) *Address {
	type addresser interface {
		Address() module.Address
	}

	if a, ok := addr.(addresser); ok {
		addr = ToAddress(a.Address())
	}

	if a, ok := addr.(module.Address); ok {
		return AddressToPtr(a)
	}
	return nil
}
