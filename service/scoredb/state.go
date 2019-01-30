package scoredb

import (
	"github.com/icon-project/goloop/module"
	"math/big"
)

const (
	ArrayDBPrefix = 0x00
	DictDBPrefix  = 0x01
	VarDBPrefix   = 0x02
)

type StateStore interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) error
	DeleteValue(key []byte) error
}

type BytesStore interface {
	Bytes() []byte
	SetBytes([]byte) error
	Delete() error
}

type Value interface {
	BigInt() *big.Int
	Int64() int64
	Address() module.Address
	Bytes() []byte
	String() string
}

type WritableValue interface {
	Value
	Delete() error
	Set(interface{}) error
}
