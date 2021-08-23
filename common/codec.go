package common

import (
	"math/big"
	"strconv"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	TypeAddress = iota + codec.TypeCustom
	TypeInt
	TypeFloat
)

type typeCodec struct{}

var TypeCodec = &typeCodec{}

func (*typeCodec) Decode(tag uint8, data []byte) (interface{}, error) {
	switch tag {
	case TypeAddress:
		if addr, err := NewAddress(data); err != nil {
			return nil, errors.CriticalFormatError.Errorf(
				"InvalidAddressBytes(bytes=%#x)", data)
		} else {
			return addr, nil
		}
	case TypeInt:
		i := new(HexInt)
		i.SetBytes(data)
		return i, nil
	case TypeFloat:
		return strconv.ParseFloat(string(data), 64)
	default:
		return 0, errors.Errorf("InvalidTypeTag:%d", tag)
	}
}

func (*typeCodec) Encode(o interface{}) (uint8, []byte, error) {
	switch obj := o.(type) {
	case module.Address:
		return TypeAddress, obj.Bytes(), nil
	case *big.Int:
		return TypeInt, intconv.BigIntToBytes(obj), nil
	case *HexInt:
		return TypeInt, obj.Bytes(), nil
	case int:
		return TypeInt, intconv.Int64ToBytes(int64(obj)), nil
	case int16:
		return TypeInt, intconv.Int64ToBytes(int64(obj)), nil
	case int32:
		return TypeInt, intconv.Int64ToBytes(int64(obj)), nil
	case int64:
		return TypeInt, intconv.Int64ToBytes(obj), nil
	case uint:
		return TypeInt, intconv.Int64ToBytes(int64(obj)), nil
	case uint16:
		return TypeInt, intconv.Int64ToBytes(int64(obj)), nil
	case uint32:
		return TypeInt, intconv.Int64ToBytes(int64(obj)), nil
	case uint64:
		return TypeInt, intconv.Uint64ToBytes(obj), nil
	default:
		return 0, nil, errors.Errorf("UnknownType:%T", obj)
	}
}

func MarshalAny(c codec.Codec, obj interface{}) ([]byte, error) {
	return codec.MarshalAny(c, TypeCodec, obj)
}

func EncodeAny(obj interface{}) (*codec.TypedObj, error) {
	return codec.EncodeAny(TypeCodec, obj)
}

func MustEncodeAny(obj interface{}) *codec.TypedObj {
	if tobj, err := codec.EncodeAny(TypeCodec, obj); err != nil {
		log.Panicf("Fail on codec.EncodeAny() err=%+v", err)
		return nil
	} else {
		return tobj
	}
}

func DecodeAny(o *codec.TypedObj) (interface{}, error) {
	return codec.DecodeAny(TypeCodec, o)
}

func DecodeAsString(o *codec.TypedObj, s string) string {
	if o != nil && o.Type == codec.TypeString {
		return o.Object.(string)
	}
	return s
}

func MustDecodeAny(o *codec.TypedObj) interface{} {
	if obj, err := codec.DecodeAny(TypeCodec, o); err != nil {
		log.Panicf("Fail on codec.DecodeAny() err=%+v", err)
		return nil
	} else {
		return obj
	}
}

func UnmarshalAny(c codec.Codec, bs []byte) (interface{}, error) {
	return codec.UnmarshalAny(c, TypeCodec, bs)
}

func DecodeAnyForJSON(o *codec.TypedObj) (interface{}, error) {
	value, err := codec.DecodeAny(TypeCodec, o)
	if err != nil {
		return value, err
	}
	return AnyForJSON(value)
}

func AnyForJSON(o interface{}) (interface{}, error) {
	switch obj := o.(type) {
	case []byte:
		return HexBytes(obj), nil
	case bool:
		b := new(HexInt)
		if obj {
			b.SetBytes(codec.TrueBytes)
		} else {
			b.SetBytes(codec.FalseBytes)
		}
		return b, nil
	case []interface{}:
		l := make([]interface{}, len(obj))
		for i, o := range obj {
			if co, err := AnyForJSON(o); err != nil {
				return nil, err
			} else {
				l[i] = co
			}
		}
		return l, nil
	case map[string]interface{}:
		m := make(map[string]interface{})
		for k, o := range obj {
			if co, err := AnyForJSON(o); err != nil {
				return nil, err
			} else {
				m[k] = co
			}
		}
		return m, nil
	default:
		return obj, nil
	}
}
