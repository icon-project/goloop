package service

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	mp "github.com/ugorji/go/codec"
	"log"
	"math/big"
)

type accountSnapshot struct {
	balance     common.HexInt
	fIsContract bool
	store       trie.Immutable
	database    db.Database
}

func (s *accountSnapshot) getBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountSnapshot) isContract() bool {
	return s.fIsContract
}

func (s *accountSnapshot) Bytes() []byte {
	b, err := codec.MP.MarshalToBytes(s)
	if err != nil {
		panic(err)
	}
	return b
}

func (s *accountSnapshot) CodecEncodeSelf(e *mp.Encoder) {
	e.Encode(s.balance)
	e.Encode(s.fIsContract)
	e.Encode(s.store.Hash())
}

func (s *accountSnapshot) CodecDecodeSelf(d *mp.Decoder) {
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

func (s *accountSnapshot) Reset(database db.Database, data []byte) error {
	s.database = database
	_, err := codec.MP.UnmarshalFromBytes(data, s)
	return err
}

func (s *accountSnapshot) Flush() error {
	if sp, ok := s.store.(trie.Snapshot); ok {
		return sp.Flush()
	}
	return nil
}

func (s *accountSnapshot) isEmpty() bool {
	return s.balance.BitLen() == 0 && s.store == nil
}

func (s *accountSnapshot) Equal(object trie.Object) bool {
	if s2, ok := object.(*accountSnapshot); ok {
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
		log.Panicf("Replacing accountSnapshot with other object(%T)", object)
	}
	return false
}

func (s *accountSnapshot) getValue(k []byte) ([]byte, error) {
	return s.store.Get(k)
}

type accountState struct {
	database    db.Database
	balance     common.HexInt
	fIsContract bool
	store       trie.Mutable
}

func (s *accountState) getBalance() *big.Int {
	v := new(big.Int)
	v.Set(&s.balance.Int)
	return v
}

func (s *accountState) setBalance(v *big.Int) {
	s.balance.Set(v)
}

func (s *accountState) isContract() bool {
	return s.fIsContract
}

func (s *accountState) getSnapshot() *accountSnapshot {
	var store trie.Immutable
	if s.store != nil {
		store = s.store.GetSnapshot()
		if store.Empty() {
			store = nil
		}
	}
	return &accountSnapshot{
		balance:     s.balance,
		fIsContract: s.fIsContract,
		store:       store,
	}
}

func (s *accountState) reset(snapshot *accountSnapshot) error {
	s.balance.Set(&snapshot.balance.Int)
	s.fIsContract = snapshot.fIsContract
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
			log.Panicf("Fail to make accountState err=%v", err)
		}
	}
	return nil
}

func (s *accountState) getValue(k []byte) ([]byte, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.Get(k)
}

func (s *accountState) setValue(k, v []byte) error {
	if s.store == nil {
		s.store = trie_manager.NewMutable(s.database, nil)
	}
	return s.store.Set(k, v)
}

func (s *accountState) deleteValue(k []byte) error {
	if s.store == nil {
		return nil
	}
	return s.store.Delete(k)
}

func newAccountState(database db.Database, snapshot *accountSnapshot) *accountState {
	s := new(accountState)
	s.database = database
	if snapshot != nil {
		s.reset(snapshot)
	}
	return s
}
