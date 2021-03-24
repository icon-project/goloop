package trie

import (
	"reflect"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
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
		Filter(prefix []byte) Iterator
		Equal(immutable Immutable, exact bool) bool
		Prove(k []byte, p [][]byte) ([]byte, error)
		Resolve(builder merkle.Builder)
		ClearCache()
		Database() db.Database
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
		Set(k, v []byte) ([]byte, error)
		Delete(k []byte) ([]byte, error)
		GetSnapshot() Snapshot
		Reset(d Immutable) error
		ClearCache()
		Database() db.Database
	}

	Object interface {
		Bytes() []byte
		Reset(s db.Database, k []byte) error
		Flush() error
		Equal(Object) bool
		Resolve(builder merkle.Builder) error
		ClearCache()
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
		Filter(prefix []byte) IteratorForObject
		Equal(object ImmutableForObject, exact bool) bool
		Prove(k []byte, p [][]byte) (Object, error)
		Resolve(builder merkle.Builder)
		ClearCache()
		Database() db.Database
	}

	SnapshotForObject interface {
		ImmutableForObject
		Flush() error
	}

	MutableForObject interface {
		Get(k []byte) (Object, error)
		Set(k []byte, o Object) (Object, error)
		Delete(k []byte) (Object, error)
		GetSnapshot() SnapshotForObject
		Reset(s ImmutableForObject)
		ClearCache()
		Database() db.Database
	}

	Manager interface {
		NewImmutable(rootHash []byte) Immutable
		NewMutable(rootHash []byte) Mutable
		NewImmutableForObject(h []byte, t reflect.Type) ImmutableForObject
		NewMutableForObject(h []byte, t reflect.Type) MutableForObject
	}
)
