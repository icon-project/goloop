package state

import (
	"bytes"
	"fmt"
	"log"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	ugorji "github.com/ugorji/go/codec"
)

type validator struct {
	pub  []byte
	addr *common.Address
}

func (v *validator) CodecEncodeSelf(e *ugorji.Encoder) {
	if len(v.pub) == 0 {
		e.Encode(v.addr)
	} else {
		e.Encode(v.pub)
	}
}

func (v *validator) CodecDecodeSelf(d *ugorji.Decoder) {
	var bs []byte
	d.Decode(&bs)
	if len(bs) == common.AddressBytes {
		v.addr = common.NewAddress(bs)
	} else {
		v.setPublicKey(bs)
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

func (v *validator) SetBytes(bs []byte) error {
	_, err := codec.MP.UnmarshalFromBytes(bs, v)
	return err
}

func (v *validator) Equal(v2 module.Validator) bool {
	return v2.Address().Equal(v.addr) && bytes.Equal(v2.PublicKey(), v.pub)
}

func (v *validator) String() string {
	return fmt.Sprintf("Validator[addr=%v,pkey=<%x>]", v.addr, v.pub)
}

func ValidatorFromAddress(a module.Address) (module.Validator, error) {
	if a == nil {
		return nil, errors.ErrIllegalArgument
	}
	if a.IsContract() {
		return nil, errors.ErrIllegalArgument
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

func validatorFromValidator(v module.Validator) (*validator, error) {
	if v == nil {
		return nil, nil
	}
	if vo, ok := v.(*validator); ok {
		return vo, nil
	} else {
		vo = new(validator)
		if err := vo.SetBytes(v.Bytes()); err != nil {
			return nil, err
		}
		return vo, nil
	}
}
