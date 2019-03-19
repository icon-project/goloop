package state

import (
	"log"
	"reflect"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/pkg/errors"
)

// WorldSnapshot represents snapshot of WorldState.
// It can be use to WorldState recover state of WorldState to at some point.
type WorldSnapshot interface {
	GetAccountSnapshot(id []byte) AccountSnapshot
	GetValidatorSnapshot() ValidatorSnapshot
	Flush() error
	StateHash() []byte
}

// WorldState represents world state.
// You may change
type WorldState interface {
	GetAccountState(id []byte) AccountState
	GetAccountSnapshot(id []byte) AccountSnapshot
	GetSnapshot() WorldSnapshot
	GetValidatorState() ValidatorState
	Reset(snapshot WorldSnapshot) error
	ClearCache()
}

type worldSnapshotImpl struct {
	database   db.Database
	accounts   trie.ImmutableForObject
	validators ValidatorSnapshot
}

func (ws *worldSnapshotImpl) GetValidatorSnapshot() ValidatorSnapshot {
	return ws.validators
}

func (ws *worldSnapshotImpl) StateHash() []byte {
	return ws.accounts.Hash()
}

func (ws *worldSnapshotImpl) Flush() error {
	if ass, ok := ws.accounts.(trie.SnapshotForObject); ok {
		if err := ass.Flush(); err != nil {
			return err
		}
	}
	return ws.validators.Flush()
}

func (ws *worldSnapshotImpl) GetAccountSnapshot(id []byte) AccountSnapshot {
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

type worldStateImpl struct {
	mutex sync.Mutex

	database        db.Database
	accounts        trie.MutableForObject
	mutableAccounts map[string]AccountState
	validators      ValidatorState
}

func (ws *worldStateImpl) GetValidatorState() ValidatorState {
	return ws.validators
}

func (ws *worldStateImpl) Reset(isnapshot WorldSnapshot) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	snapshot, ok := isnapshot.(*worldSnapshotImpl)
	if !ok {
		return errors.New("InvalidSnapshotType")
	}
	if ws.database != snapshot.database {
		return errors.New("InvalidSnapshotWithDifferentDB")
	}
	ws.accounts.Reset(snapshot.accounts)
	for id, as := range ws.mutableAccounts {
		key := addressIDToKey([]byte(id))
		value, err := ws.accounts.Get(key)
		if err != nil {
			log.Panicf("Fail to read account value")
		}
		if value == nil {
			as.Clear()
		} else {
			if err := as.Reset(value.(AccountSnapshot)); err != nil {
				return err
			}
		}
	}
	ws.validators.Reset(snapshot.GetValidatorSnapshot())
	return nil
}

func addressIDToKey(id []byte) []byte {
	if id == nil {
		return []byte("genesis")
	}
	return crypto.SHA3Sum256(id)
}

func (ws *worldStateImpl) GetAccountState(id []byte) AccountState {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

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

func (ws *worldStateImpl) ClearCache() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	for id, as := range ws.mutableAccounts {
		key := addressIDToKey([]byte(id))
		s := as.GetSnapshot()
		if s.Empty() {
			if err := ws.accounts.Delete(key); err != nil {
				log.Panicf("Fail to delete account key = %x", key)
			}
		} else {
			if err := ws.accounts.Set(key, s); err != nil {
				log.Panicf("Fail to set snapshot for %x", key)
			}
		}
	}
	ws.mutableAccounts = make(map[string]AccountState)
}

func (ws *worldStateImpl) GetAccountSnapshot(id []byte) AccountSnapshot {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if a, ok := ws.mutableAccounts[string(id)]; ok {
		return a.GetSnapshot()
	}

	key := addressIDToKey(id)
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Panicf("Fail to get acount for %x err=%+v", key, err)
		return nil
	}
	if obj != nil {
		return obj.(*accountSnapshotImpl)
	}

	ass := new(accountSnapshotImpl)
	ass.database = ws.database
	return ass
}

func (ws *worldStateImpl) GetSnapshot() WorldSnapshot {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	for id, as := range ws.mutableAccounts {
		key := addressIDToKey([]byte(id))
		s := as.GetSnapshot()
		if s.Empty() {
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
		database:   ws.database,
		accounts:   ws.accounts.GetSnapshot(),
		validators: ws.validators.GetSnapshot(),
	}
}

func NewWorldState(database db.Database, stateHash []byte, vs ValidatorSnapshot) WorldState {
	ws := new(worldStateImpl)
	ws.database = database
	ws.accounts = trie_manager.NewMutableForObject(database, stateHash, reflect.TypeOf((*accountSnapshotImpl)(nil)))
	ws.mutableAccounts = make(map[string]AccountState)
	if vs == nil {
		ws.validators, _ = ValidatorStateFromHash(database, nil)
	} else {
		ws.validators = ValidatorStateFromSnapshot(vs)
	}
	return ws
}

func NewWorldSnapshot(dbase db.Database, stateHash []byte, vs ValidatorSnapshot) WorldSnapshot {
	ws := new(worldSnapshotImpl)
	ws.database = dbase
	ws.accounts = trie_manager.NewImmutableForObject(dbase, stateHash,
		reflect.TypeOf((*accountSnapshotImpl)(nil)))
	if vs == nil {
		vs, _ = ValidatorSnapshotFromHash(dbase, nil)
	}
	ws.validators = vs

	return ws
}

func WorldStateFromSnapshot(wss WorldSnapshot) (WorldState, error) {
	if wss, ok := wss.(*worldSnapshotImpl); ok {
		ws := new(worldStateImpl)
		ws.database = wss.database
		ws.accounts = trie_manager.NewMutableFromImmutableForObject(wss.accounts)
		ws.mutableAccounts = make(map[string]AccountState)
		ws.validators = ValidatorStateFromSnapshot(wss.GetValidatorSnapshot())
		return ws, nil
	}
	return nil, common.ErrIllegalArgument
}
