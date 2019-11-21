package common

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"gopkg.in/vmihailenco/msgpack.v4"
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
		if err := a.SetTypeAndID(isContract, bytes); err != nil {
			return err
		}
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
	if len(b) == 0 {
		return ErrIllegalArgument
	}
	switch b[0] {
	case 0:
		return a.SetTypeAndID(false, b[1:])
	case 1:
		return a.SetTypeAndID(true, b[1:])
	default:
		return ErrIllegalArgument
	}
}

func (a *Address) SetTypeAndID(ic bool, id []byte) error {
	if id == nil {
		return ErrIllegalArgument
	}
	switch {
	case len(id) < AddressIDBytes:
		copy(a[AddressIDBytes-len(id)+1:], id)
	default:
		copy(a[1:], id)
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
	a.SetTypeAndID(false, b)
	return a
}

func NewAddress(b []byte) *Address {
	a := new(Address)
	a.SetBytes(b)
	return a
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
	a := new(Address)
	pk := pubKey.SerializeUncompressed()
	if pk == nil {
		log.Panicln("FAIL invalid public key:", pubKey)
	}
	digest := crypto.SHA3Sum256(pk[1:])
	a.SetTypeAndID(false, digest[len(digest)-20:])
	return a
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

func (a Address) EncodeMsgpack(e *msgpack.Encoder) error {
	return e.EncodeBytes([]byte(a[:]))
}

func (a *Address) DecodeMsgpack(d *msgpack.Decoder) error {
	if bs, err := d.DecodeBytes(); err != nil {
		return err
	} else {
		return a.SetBytes(bs)
	}
}
