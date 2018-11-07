package service

import (
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"log"
	"reflect"
)

type worldSnapshot struct {
	accounts trie.SnapshotForObject
}

type worldState struct {
	database        db.Database
	accounts        trie.MutableForObject
	mutableAccounts map[string]*accountState
}

func (ws *worldState) reset(snapshot *worldSnapshot) {
	ws.accounts.Reset(snapshot.accounts)
	ws.mutableAccounts = make(map[string]*accountState)
}

func (ws *worldSnapshot) stateHash() []byte {
	return ws.accounts.Hash()
}

func (ws *worldSnapshot) flush() error {
	return ws.accounts.Flush()
}

func (ws *worldSnapshot) getAccountSnapshot(id []byte) *accountSnapshot {
	key := addressIDToKey(id)
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Panicf("Fail to get acount for %x err=%v", key, err)
		return nil
	}
	if obj == nil {
		return nil
	}
	if s, ok := obj.(*accountSnapshot); ok {
		return s
	} else {
		log.Panicf("Returned account isn't accountSnapshot type=%T", obj)
		return nil
	}
}

func addressIDToKey(id []byte) []byte {
	return crypto.SHA3Sum256(id)
}

func (ws *worldState) getAccountState(id []byte) *accountState {
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
	var as *accountSnapshot
	if obj != nil {
		as = obj.(*accountSnapshot)
	}
	ac := newAccountState(ws.database, as)
	ws.mutableAccounts[ids] = ac
	return ac
}

func (ws *worldState) getSnapshot() *worldSnapshot {
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
	return &worldSnapshot{
		accounts: ws.accounts.GetSnapshot(),
	}
}

func NewWorldState(database db.Database, stateHash []byte) *worldState {
	ws := new(worldState)
	ws.accounts = trie_manager.NewMutableForObject(database, stateHash, reflect.TypeOf((*accountSnapshot)(nil)))
	ws.mutableAccounts = make(map[string]*accountState)
	return ws
}
