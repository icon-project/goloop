package state

import (
	"bytes"
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type validator struct {
	pub  []byte
	addr *common.Address
}

func (v *validator) RLPEncodeSelf(e codec.Encoder) error {
	if len(v.pub) == 0 {
		return e.Encode(v.addr)
	} else {
		return e.Encode(v.pub)
	}
}

func (v *validator) RLPDecodeSelf(d codec.Decoder) error {
	bs, err := d.DecodeBytes()
	if err != nil {
		return err
	}
	if len(bs) == common.AddressBytes {
		if addr, err := common.NewAddress(bs); err != nil {
			return err
		} else {
			v.addr = addr
		}
		return nil
	} else {
		return v.setPublicKey(bs)
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
	bytes, err := codec.BC.MarshalToBytes(v)
	if err != nil {
		log.Errorf("Fail to convert validator to bytes. err=%+v\n", err)
		return nil
	}
	return bytes
}

func (v *validator) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, v)
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
		addr: common.AddressToPtr(a),
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
