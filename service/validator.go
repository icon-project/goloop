package service

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
	"log"
)

type validator struct {
	pub  []byte
	addr *common.Address
}

func (v *validator) CodecEncodeSelf(e *ugorji.Encoder) {
	if len(v.pub) == 0 {
		e.Encode([]byte{})
		e.Encode(v.addr)
	} else {
		e.Encode(v.pub)
	}
}

func (v *validator) CodecDecodeSelf(d *ugorji.Decoder) {
	var pubkey []byte
	d.Decode(&pubkey)
	if len(pubkey) == 0 {
		d.Decode(&v.addr)
	} else {
		v.setPublicKey(pubkey)
	}
}

func (v *validator) setPublicKey(bytes []byte) error {
	pk, err := crypto.ParsePublicKey(bytes)
	if err != nil {
		return err
	}
	v.pub = pk.SerializeCompressed()
	v.addr = common.NewAccountAddressFromPublicKey(pk)
	return nil
}

func (v *validator) Address() module.Address {
	return v.addr
}

func (v *validator) PublicKey() []byte {
	return v.pub
}

func (v *validator) Bytes() []byte {
	bytes, err := codec.MP.MarshalToBytes(v)
	if err != nil {
		log.Panicf("Fail to convert validator to bytes")
		return nil
	}
	return bytes
}

func (v *validator) String() string {
	return fmt.Sprintf("Validator[addr=%v,pkey=<%x>]", v.addr, v.pub)
}

func ValidatorFromAddress(a module.Address) (module.Validator, error) {
	if a == nil {
		return nil, errors.New("IllegalArgument")
	}
	v := &validator{
		pub:  nil,
		addr: new(common.Address),
	}
	err := v.addr.SetBytes(a.Bytes())
	if err != nil {
		return nil, err
	}
	return v, nil
}

func ValidatorFromPublicKey(pk []byte) (module.Validator, error) {
	v := new(validator)
	if err := v.setPublicKey(pk); err != nil {
		return nil, err
	}
	return v, nil
}
