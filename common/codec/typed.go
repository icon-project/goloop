package codec

import (
	ugorji "github.com/ugorji/go/codec"
)

const (
	TypeNil uint8 = iota
	TypeDict
	TypeList
	TypeBytes
	TypeString
	TypeCustom = 10
)

type TypeCodec interface {
	Decode(tag uint8, data []byte) (interface{}, error)
	Encode(o interface{}) (uint8, []byte, error)
}

type typedObjBase struct {
	Type   uint8
	Object interface{}
}

type typedObjDummy struct {
	Type   uint8
	Object ugorji.Raw
}

type TypedObj struct {
	typedObjBase
}

func (o *TypedObj) CodecEncodeSelf(e *ugorji.Encoder) {
	e.Encode(&o.typedObjBase)
}

func (o *TypedObj) CodecDecodeSelf(d *ugorji.Decoder) {
	var tmp typedObjDummy
	d.Decode(&tmp)
	o.Type = tmp.Type
	switch o.Type {
	case TypeNil:
		return
	case TypeDict:
		var m map[string]*TypedObj
		MP.UnmarshalFromBytes([]byte(tmp.Object), &m)
		o.Object = m
	case TypeList:
		var l []*TypedObj
		MP.UnmarshalFromBytes([]byte(tmp.Object), &l)
		o.Object = l
	case TypeString:
		var s string
		MP.UnmarshalFromBytes([]byte(tmp.Object), &s)
		o.Object = s
	default:
		var bs []byte
		MP.UnmarshalFromBytes([]byte(tmp.Object), &bs)
		o.Object = bs
	}
}

func newTypedObj(t uint8, o interface{}) *TypedObj {
	return &TypedObj{typedObjBase{t, o}}
}

func MarshalAny(tc TypeCodec, o interface{}) ([]byte, error) {
	if ao, err := EncodeAny(tc, o); err != nil {
		return nil, err
	} else {
		return MarshalToBytes(ao)
	}
}

func EncodeAny(tc TypeCodec, o interface{}) (*TypedObj, error) {
	if o == nil {
		return newTypedObj(TypeNil, nil), nil
	}
	switch obj := o.(type) {
	case string:
		return newTypedObj(TypeString, obj), nil
	case []byte:
		return newTypedObj(TypeBytes, obj), nil
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
	case map[string]interface{}:
		m := make(map[string]*TypedObj)
		for k, o := range obj {
			if eo, err := EncodeAny(tc, o); err != nil {
				return nil, err
			} else {
				m[k] = eo
			}
		}
		return newTypedObj(TypeDict, m), nil
	case map[string]int64:
		m := make(map[string]*TypedObj)
		for k, o := range obj {
			if eo, err := EncodeAny(tc, o); err != nil {
				return nil, err
			} else {
				m[k] = eo
			}
		}
		return newTypedObj(TypeDict, m), nil
	case *TypedObj:
		return obj, nil
	default:
		if tag, bytes, err := tc.Encode(obj); err != nil {
			return nil, err
		} else {
			return newTypedObj(tag, bytes), nil
		}
	}
}

func UnmarshalAny(tc TypeCodec, bs []byte) (interface{}, error) {
	var to TypedObj
	if _, err := UnmarshalFromBytes(bs, &to); err != nil {
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
	case TypeDict:
		m := make(map[string]interface{})
		for k, nto := range to.Object.(map[string]*TypedObj) {
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
