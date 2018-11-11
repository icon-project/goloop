package trie

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
)

type (
	/*
	 */
	Immutable interface {
		Empty() bool
		// Get returns the value stored under the specified key, or nil if the key doesn't exist.
		Get(k []byte) ([]byte, error)
		Hash() []byte               // return nil if this Tree is empty
		GetProof(k []byte) [][]byte // return nill of this Tree is empty
		Iterator() Iterator
		Equal(immutable Immutable, exact bool) bool
	}

	Snapshot interface {
		Immutable
		Flush() error
	}

	Iterator interface {
		Next() error
		Has() bool
		Get() (value []byte, key []byte, err error)
	}

	Mutable interface {
		Get(k []byte) ([]byte, error)
		Set(k, v []byte) error
		Delete(k []byte) error
		GetSnapshot() Snapshot
		Reset(d Immutable) error
	}

	Object interface {
		Bytes() []byte
		Reset(s db.Database, k []byte) error
		Flush() error
		Equal(Object) bool
	}

	IteratorForObject interface {
		Next() error
		Has() bool
		Get() (Object, []byte, error)
	}

	ImmutableForObject interface {
		Empty() bool
		Get(k []byte) (Object, error)
		Hash() []byte
		GetProof(k []byte) [][]byte // return nill of this Tree is empty
		Iterator() IteratorForObject
		Equal(object ImmutableForObject, exact bool) bool
	}

	SnapshotForObject interface {
		ImmutableForObject
		Flush() error
	}

	MutableForObject interface {
		Get(k []byte) (Object, error)
		Set(k []byte, o Object) error
		Delete(k []byte) error
		GetSnapshot() SnapshotForObject
		Reset(s ImmutableForObject)
	}

	Manager interface {
		NewImmutable(rootHash []byte) Immutable
		NewMutable(rootHash []byte) Mutable
		NewImmutableForObject(h []byte, t reflect.Type) ImmutableForObject
		NewMutableForObject(h []byte, t reflect.Type) MutableForObject
	}
)

// Verify proofs,
//func Verify(key []byte, proofs [][]byte, rootHash []byte ) bool {
//	return true
//}
