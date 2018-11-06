package service

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/mpt"
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

func accountNameToKey(s string) []byte {
	return []byte(s)
}

func (ws *worldState) getAccountState(name string) *accountState {
	if a, ok := ws.mutableAccounts[name]; ok {
		return a
	}
	key := accountNameToKey(name)
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Panicf("Fail to get acount for %x", key)
		return nil
	}
	ac := newAccountState(ws.database, obj.(*accountSnapshot))
	ws.mutableAccounts[name] = ac
	return ac
}

func (ws *worldState) getSnapshot() *worldSnapshot {
	for name, as := range ws.mutableAccounts {
		key := accountNameToKey(name)
		s := as.getSnapshot()
		if s.isEmpty() {
			if err := ws.accounts.Delete(key); err != nil {
				log.Panicf("Fail to delete account key = %x", key)
			}
		} else {
			if err := ws.accounts.Set(key, as.getSnapshot()); err != nil {
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
	ws.accounts = mpt.NewMutableForObject(database, stateHash, reflect.TypeOf((*accountSnapshot)(nil)))
	ws.mutableAccounts = make(map[string]*accountState)
	return ws
}
