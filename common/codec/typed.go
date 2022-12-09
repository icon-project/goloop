package codec

import (
	"bytes"

	"github.com/icon-project/goloop/common/errors"
)

const (
	TypeNil uint8 = iota
	TypeDict
	TypeList
	TypeBytes
	TypeString
	TypeBool
	TypeCustom = 10
)

var (
	TrueBytes  = []byte{0x01}
	FalseBytes = []byte{0x00}
)

type TypeCodec interface {
	Decode(tag uint8, data []byte) (interface{}, error)
	Encode(o interface{}) (uint8, []byte, error)
}

type typedObjBase struct {
	Type   uint8
	Object interface{}
}

type TypedObj struct {
	typedObjBase
}

var Nil = &TypedObj{
	typedObjBase{
		TypeNil, nil,
	},
}

func (o *TypedObj) RLPEncodeSelf(e Encoder) error {
	return e.Encode(o.typedObjBase)
}

func (o *TypedObj) RLPDecodeSelf(d Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var t uint8
	if err := d2.Decode(&t); err != nil {
		return err
	}
	o.Type = t
	switch t {
	case TypeNil:
	case TypeDict:
		var m *TypedDict
		err = d2.Decode(&m)
		if err != nil {
			return err
		}
		o.Object = m
	case TypeList:
		var l []*TypedObj
		err = d2.Decode(&l)
		if err != nil {
			return err
		}
		o.Object = l
	case TypeString:
		var s string
		if err := d2.Decode(&s); err != nil {
			return err
		}
		o.Object = s
	case TypeBool:
		var bs []byte
		if err := d2.Decode(&bs); err != nil {
			return err
		}
		o.Object = bs
	default:
		var bs []byte
		if err := d2.Decode(&bs); err != nil {
			return err
		}
		o.Object = bs
	}
	return nil
}

func newTypedObj(t uint8, o interface{}) *TypedObj {
	return &TypedObj{typedObjBase{t, o}}
}

func MarshalAny(c Codec, tc TypeCodec, o interface{}) ([]byte, error) {
	if ao, err := EncodeAny(tc, o); err != nil {
		return nil, err
	} else {
		return c.MarshalToBytes(ao)
	}
}

func EncodeAny(tc TypeCodec, o interface{}) (*TypedObj, error) {
	if o == nil {
		return Nil, nil
	}
	switch obj := o.(type) {
	case string:
		return newTypedObj(TypeString, obj), nil
	case []byte:
		return newTypedObj(TypeBytes, obj), nil
	case bool:
		if obj {
			return newTypedObj(TypeBool, TrueBytes), nil
		} else {
			return newTypedObj(TypeBool, FalseBytes), nil
		}
	case []interface{}:
		l := make([]*TypedObj, len(obj))
		for i, o := range obj {
			if eo, err := EncodeAny(tc, o); err != nil {
				return nil, err
			} else {
				l[i] = eo
			}
		}
		return newTypedObj(TypeList, l), nil
	case []*TypedObj:
		return newTypedObj(TypeList, obj), nil
	case map[string]interface{}:
		m := make(map[string]*TypedObj)
		for k, o := range obj {
			if eo, err := EncodeAny(tc, o); err != nil {
				return nil, err
			} else {
				m[k] = eo
			}
		}
		return newTypedObj(TypeDict, &TypedDict{Map: m}), nil
	case map[string]*TypedObj:
		return newTypedObj(TypeDict, &TypedDict{Map: obj}), nil
	case *TypedObj:
		return obj, nil
	case *TypedDict:
		return newTypedObj(TypeDict, obj), nil
	default:
		if tag, bytes, err := tc.Encode(obj); err != nil {
			return nil, err
		} else {
			return newTypedObj(tag, bytes), nil
		}
	}
}

func UnmarshalAny(c Codec, tc TypeCodec, bs []byte) (interface{}, error) {
	var to TypedObj
	if _, err := c.UnmarshalFromBytes(bs, &to); err != nil {
		return nil, err
	}
	return DecodeAny(tc, &to)
}

func DecodeAny(tc TypeCodec, to *TypedObj) (interface{}, error) {
	if to == nil {
		return nil, nil
	}
	switch to.Type {
	case TypeNil:
		return nil, nil
	case TypeString, TypeBytes:
		return to.Object, nil
	case TypeBool:
		bs := to.Object.([]byte)
		if bytes.Equal(bs, FalseBytes) {
			return false, nil
		} else if bytes.Equal(bs, TrueBytes) {
			return true, nil
		} else {
			return nil, errors.ErrIllegalArgument
		}
	case TypeDict:
		m := make(map[string]interface{})
		for k, nto := range to.Object.(*TypedDict).Map {
			var err error
			m[k], err = DecodeAny(tc, nto)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	case TypeList:
		tol := to.Object.([]*TypedObj)
		l := make([]interface{}, len(tol))
		for i, to := range tol {
			var err error
			l[i], err = DecodeAny(tc, to)
			if err != nil {
				return nil, err
			}
		}
		return l, nil
	default:
		bs := to.Object.([]byte)
		obj, err := tc.Decode(to.Type, bs)
		if err != nil {
			return nil, err
		} else {
			return obj, nil
		}
	}
}
