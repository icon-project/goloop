package scoredb

import (
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
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

type ReadOnlyStore interface {
	GetValue(key []byte) ([]byte, error)
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

type readonlyStateStore struct {
	ReadOnlyStore
}

func (*readonlyStateStore) SetValue(key []byte, value []byte) error {
	return errors.InvalidStateError.New("SetValue() on ReadOnlyStore")
}

func (*readonlyStateStore) DeleteValue(key []byte) error {
	return errors.InvalidStateError.New("DeleteValue() on ReadOnlyStore")
}

func NewStateStoreWith(s ReadOnlyStore) StateStore {
	if s == nil {
		return nil
	}
	return &readonlyStateStore{s}
}
