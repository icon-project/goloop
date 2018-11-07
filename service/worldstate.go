package service

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/pkg/errors"
	"log"
	"reflect"
)

type worldSnapshot interface {
	getAccountSnapshot(id []byte) accountSnapshot
	flush() error
	stateHash() []byte
}

type worldState interface {
	getAccountState(id []byte) accountState
	getSnapshot() worldSnapshot
	reset(snapshot worldSnapshot) error
}

type worldSnapshotImpl struct {
	accounts trie.SnapshotForObject
}

type worldStateImpl struct {
	database        db.Database
	accounts        trie.MutableForObject
	mutableAccounts map[string]accountState
}

func (ws *worldStateImpl) reset(isnapshot worldSnapshot) error {
	snapshot, ok := isnapshot.(*worldSnapshotImpl)
	if !ok {
		return errors.New("InvalidSnapshotType")
	}
	ws.accounts.Reset(snapshot.accounts)
	ws.mutableAccounts = make(map[string]accountState)
	return nil
}

func (ws *worldSnapshotImpl) stateHash() []byte {
	return ws.accounts.Hash()
}

func (ws *worldSnapshotImpl) flush() error {
	return ws.accounts.Flush()
}

func (ws *worldSnapshotImpl) getAccountSnapshot(id []byte) accountSnapshot {
	key := addressIDToKey(id)
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Panicf("Fail to get acount for %x err=%v", key, err)
		return nil
	}
	if obj == nil {
		return nil
	}
	if s, ok := obj.(*accountSnapshotImpl); ok {
		return s
	} else {
		log.Panicf("Returned account isn't accountSnapshotImpl type=%T", obj)
		return nil
	}
}

func addressIDToKey(id []byte) []byte {
	return crypto.SHA3Sum256(id)
}

func (ws *worldStateImpl) getAccountState(id []byte) accountState {
	ids := string(id)
	if a, ok := ws.mutableAccounts[ids]; ok {
		return a
	}
	key := addressIDToKey(id)
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Panicf("Fail to get acount for %x err=%+v", key, err)
		return nil
	}
	var as *accountSnapshotImpl
	if obj != nil {
		as = obj.(*accountSnapshotImpl)
	}
	ac := newAccountState(ws.database, as)
	ws.mutableAccounts[ids] = ac
	return ac
}

func (ws *worldStateImpl) getSnapshot() worldSnapshot {
	for id, as := range ws.mutableAccounts {
		key := addressIDToKey([]byte(id))
		s := as.getSnapshot()
		if s.isEmpty() {
			if err := ws.accounts.Delete(key); err != nil {
				log.Panicf("Fail to delete account key = %x", key)
			}
		} else {
			if err := ws.accounts.Set(key, s); err != nil {
				log.Panicf("Fail to set snapshot for %x", key)
			}
		}
	}
	return &worldSnapshotImpl{
		accounts: ws.accounts.GetSnapshot(),
	}
}

func NewWorldState(database db.Database, stateHash []byte) worldState {
	ws := new(worldStateImpl)
	ws.accounts = trie_manager.NewMutableForObject(database, stateHash, reflect.TypeOf((*accountSnapshotImpl)(nil)))
	ws.mutableAccounts = make(map[string]accountState)
	return ws
}
