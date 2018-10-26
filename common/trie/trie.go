package trie

import (
	"github.com/icon-project/goloop/common/db"
)

type (
	/*
	 */
	Immutable interface {
		// Returns the value to which the specified key is mapped, or nil if this Tree has no mapping for the key
		Get(k []byte) ([]byte, error)
		RootHash() []byte        // return nil if this Tree is empty
		Proof(k []byte) [][]byte // return nill of this Tree is empty
		// TODO: have to implement prove function with returned value from Proof()
		// 			but the prove funcion don't have to be interface
	}

	Snapshot interface {
		Immutable
		Flush() error
	}

	// TODO : need Cache??
	Cache interface {
		Immutable
		Load(db db.DB, root []byte) error
	}

	Mutable interface {
		Immutable
		Set(k, v []byte) error
		Delete(k []byte) error
		GetSnapshot() Snapshot
		Reset(d Immutable) error
	}

	Manager interface {
		NewImmutable(rootHash []byte) Immutable
		NewMutable(rootHash []byte) Mutable
	}

	Object interface {
		Bytes() []byte
		Reset(s db.Store, k []byte) error
		Flush() error
		Equal(Object) bool
	}

	ImmutableForObject interface {
		Get(k []byte) (Object, error)
		GetBytes(k []byte) ([]byte, error)
		Hash() []byte
	}

	SnapshotForObject interface {
		ImmutableForObject
		Flush() error
	}

	MutableForObject interface {
		Set(k []byte, o Object) error
		Delete(k []byte) error
		GetSnapshot() SnapshotForObject
		Reset(s ImmutableForObject)
	}
)

func Verify(proofs [][]byte, db db.DB) bool {
	return true
}
