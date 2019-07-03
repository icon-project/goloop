package scoredb

import (
	"github.com/icon-project/goloop/common/log"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
)

func ToKey(prefix byte, keys ...interface{}) []byte {
	return AppendKeys([]byte{prefix}, keys...)
}

func rlpCountBytesForSize(b int) int {
	var cnt = int(1)
	for b >>= 8; b > 0; cnt++ {
		b >>= 8
	}
	return cnt
}

func rlpEncodeBytes(b []byte) []byte {
	var blen = len(b)
	if blen == 1 && b[0] < 0x80 {
		return b
	}
	if blen <= 55 {
		buf := make([]byte, blen+1)
		buf[0] = byte(0x80 + blen)
		copy(buf[1:], b)
		return buf
	}
	tslen := rlpCountBytesForSize(blen)
	buf := make([]byte, 1+tslen+blen)
	buf[0] = byte(0x80 + 55 + tslen)
	for tsidx := tslen; tsidx > 0; tsidx-- {
		buf[tsidx] = byte(blen & 0xff)
		blen >>= 8
	}
	copy(buf[tslen+1:], b)
	return buf
}

func AppendKeys(key []byte, keys ...interface{}) []byte {
	list := make([][]byte, len(keys))
	size := len(key)
	for i, k := range keys {
		list[i] = rlpEncodeBytes(ToBytes(k))
		size += len(list[i])
	}
	kbytes := make([]byte, len(key), size)
	copy(kbytes, key)
	for _, k := range list {
		kbytes = append(kbytes, k...)
	}
	return kbytes
}

func ToBytes(v interface{}) []byte {
	switch obj := v.(type) {
	case Value:
		return obj.Bytes()
	case module.Address:
		return obj.Bytes()
	case bool:
		if obj {
			return []byte{1}
		} else {
			return []byte{0}
		}
	case int:
		return common.Int64ToBytes(int64(obj))
	case int16:
		return common.Int64ToBytes(int64(obj))
	case int32:
		return common.Int64ToBytes(int64(obj))
	case int64:
		return common.Int64ToBytes(obj)
	case *big.Int:
		return common.BigIntToBytes(obj)
	case *common.HexInt:
		return obj.Bytes()
	case string:
		return []byte(obj)
	case []byte:
		return obj
	default:
		log.Panicf("UnknownType(%T)", v)
		return []byte{}
	}
}

type valueImpl struct {
	BytesStore
}

func (e *valueImpl) BigInt() *big.Int {
	if bs := e.Bytes(); bs != nil {
		value := new(big.Int)
		return common.BigIntSetBytes(value, bs)
	} else {
		return nil
	}
}

func (e *valueImpl) Int64() int64 {
	if bs := e.Bytes(); bs != nil {
		if len(bs) <= 8 {
			return common.BytesToInt64(bs)
		}
	}
	return 0
}

func (e *valueImpl) Address() module.Address {
	if bs := e.Bytes(); bs != nil {
		var addr common.Address
		if err := addr.SetBytes(bs); err == nil {
			return &addr
		}
	}
	return nil
}

func (e *valueImpl) String() string {
	if bs := e.Bytes(); bs != nil {
		return string(bs)
	} else {
		return ""
	}
}

func (e *valueImpl) Bool() bool {
	if bs := e.Bytes(); len(bs) > 1 || bs[0] != 0 {
		return true
	}
	return false
}

func (e *valueImpl) Set(v interface{}) error {
	bs := ToBytes(v)
	return e.SetBytes(bs)
}

type bytesEntry []byte

func (e bytesEntry) Bytes() []byte {
	return []byte(e)
}

func (e bytesEntry) SetBytes([]byte) error {
	return nil
}

func (e bytesEntry) Delete() error {
	return nil
}

func NewValueFromBytes(bs []byte) Value {
	if bs == nil {
		return nil
	}
	return &valueImpl{bytesEntry(bs)}
}

type storeEntry struct {
	key   []byte
	store StateStore
}

func (e *storeEntry) Delete() error {
	return e.store.DeleteValue(e.key)
}

func (e *storeEntry) SetBytes(bs []byte) error {
	return e.store.SetValue(e.key, bs)
}

func (e *storeEntry) Bytes() []byte {
	if bs, err := e.store.GetValue(e.key); err == nil && bs != nil {
		return bs
	} else {
		return nil
	}
}

func NewValueFromStore(store StateStore, kbytes []byte) WritableValue {
	return &valueImpl{&storeEntry{kbytes, store}}
}
