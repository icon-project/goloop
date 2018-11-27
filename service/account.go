package service

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/pkg/errors"
	ugorji "github.com/ugorji/go/codec"
	"log"
	"math/big"
)

// AccountSnapshot represents immutable account state
// It can be get from AccountState or WorldSnapshot.
type AccountSnapshot interface {
	trie.Object
	GetBalance() *big.Int
	IsContract() bool
	Empty() bool
	GetValue(k []byte) ([]byte, error)
}

// AccountState represents mutable account state.
// You may change account state with this object. It can be get from
// WorldState. Changes in this object will be retrieved by WorldState.
// Of course, it also can be changed by WorldState.
type AccountState interface {
	GetBalance() *big.Int
	IsContract() bool
	GetValue(k []byte) ([]byte, error)
	SetBalance(v *big.Int)
	SetValue(k, v []byte) error
	DeleteValue(k []byte) error
	GetSnapshot() AccountSnapshot
	Reset(snapshot AccountSnapshot) error
}

type accountSnapshotImpl struct {
	balance     common.HexUint
	fIsContract bool
	store       trie.Immutable
	database    db.Database
}

func (s *accountSnapshotImpl) GetBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountSnapshotImpl) IsContract() bool {
	return s.fIsContract
}

func (s *accountSnapshotImpl) GetValue(k []byte) ([]byte, error) {
	return s.store.Get(k)
}

func (s *accountSnapshotImpl) Empty() bool {
	return s.balance.BitLen() == 0 && s.store == nil
}

func (s *accountSnapshotImpl) Bytes() []byte {
	b, err := codec.MP.MarshalToBytes(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (s *accountSnapshotImpl) Reset(database db.Database, data []byte) error {
	s.database = database
	_, err := codec.MP.UnmarshalFromBytes(data, s)
	return err
}

func (s *accountSnapshotImpl) Flush() error {
	if sp, ok := s.store.(trie.Snapshot); ok {
		return sp.Flush()
	}
	return nil
}

func (s *accountSnapshotImpl) Equal(object trie.Object) bool {
	if s2, ok := object.(*accountSnapshotImpl); ok {
		if s == s2 {
			return true
		}
		if s == nil || s2 == nil {
			return false
		}
		if s.fIsContract != s2.fIsContract ||
			s.balance.Cmp(&s2.balance.Int) != 0 {
			return false
		}
		if s.store == s2.store {
			return true
		}
		if s.store == nil || s2.store == nil {
			return false
		}
		return s.store.Equal(s2.store, false)
	} else {
		log.Panicf("Replacing accountSnapshotImpl with other object(%T)", object)
	}
	return false
}

func (s *accountSnapshotImpl) CodecEncodeSelf(e *ugorji.Encoder) {
	e.Encode(s.balance)
	e.Encode(s.fIsContract)
	if s.store != nil {
		e.Encode(s.store.Hash())
	} else {
		e.Encode(nil)
	}
}

func (s *accountSnapshotImpl) CodecDecodeSelf(d *ugorji.Decoder) {
	if err := d.Decode(&s.balance); err != nil {
		log.Fatalf("Fail to decode balance in account")
	}
	if err := d.Decode(&s.fIsContract); err != nil {
		log.Fatalf("Fail to decode isContract in account")
	}
	var hash []byte
	if err := d.Decode(&hash); err != nil {
		log.Fatalf("Fail to decode hash in account")
	} else {
		if len(hash) == 0 {
			s.store = nil
		} else {
			s.store = trie_manager.NewImmutable(s.database, hash)
		}
	}
}

type accountStateImpl struct {
	database   db.Database
	balance    common.HexUint
	isContract bool
	store      trie.Mutable
}

func (s *accountStateImpl) GetBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountStateImpl) SetBalance(v *big.Int) {
	s.balance.Set(v)
}

func (s *accountStateImpl) IsContract() bool {
	return s.isContract
}

func (s *accountStateImpl) GetSnapshot() AccountSnapshot {
	var store trie.Immutable
	if s.store != nil {
		store = s.store.GetSnapshot()
		if store.Empty() {
			store = nil
		}
	}
	return &accountSnapshotImpl{
		balance:     s.balance.Clone(),
		fIsContract: s.isContract,
		store:       store,
	}
}

func (s *accountStateImpl) Reset(isnapshot AccountSnapshot) error {
	snapshot, ok := isnapshot.(*accountSnapshotImpl)
	if !ok {
		log.Panicf("It tries to Reset with invalid snapshot type=%T", s)
	}

	s.balance.Set(&snapshot.balance.Int)
	s.isContract = snapshot.fIsContract
	if s.store == nil && snapshot.store == nil {
		return nil
	}
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	if snapshot.store == nil {
		s.store = nil
	} else {
		if err := s.store.Reset(snapshot.store); err != nil {
			log.Panicf("Fail to make accountStateImpl err=%v", err)
		}
	}
	return nil
}

func (s *accountStateImpl) GetValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountStateImpl) SetValue(k, v []byte) error {
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	return s.store.Set(k, v)
}

func (s *accountStateImpl) DeleteValue(k []byte) error {
	if s.store == nil {
		return nil
	}
	return s.store.Delete(k)
}

func newAccountState(database db.Database, snapshot *accountSnapshotImpl) AccountState {
	s := new(accountStateImpl)
	s.database = database
	if snapshot != nil {
		s.Reset(snapshot)
	}
	return s
}

type accountROState struct {
	AccountSnapshot
}

func (a *accountROState) SetBalance(v *big.Int) {
	log.Panicf("accountROState().SetBalance() is invoked")
}

func (a *accountROState) SetValue(k, v []byte) error {
	return errors.New("ReadOnlyState")
}

func (a *accountROState) DeleteValue(k []byte) error {
	return errors.New("ReadOnlyState")
}

func (a *accountROState) GetSnapshot() AccountSnapshot {
	return a.AccountSnapshot
}

func (a *accountROState) Reset(snapshot AccountSnapshot) error {
	return errors.New("ReadOnlyState")
}

func newAccountROState(snapshot AccountSnapshot) AccountState {
	return &accountROState{snapshot}
}
