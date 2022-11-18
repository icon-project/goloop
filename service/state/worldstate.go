package state

import (
	"sync"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/cache"
	"github.com/icon-project/goloop/common/trie/trie_manager"
)

// WorldSnapshot represents snapshot of WorldState.
// It can be use to WorldState recover state of WorldState to at some point.
type WorldSnapshot interface {
	GetAccountSnapshot(id []byte) AccountSnapshot
	GetValidatorSnapshot() ValidatorSnapshot
	GetExtensionSnapshot() ExtensionSnapshot
	GetBTPSnapshot() BTPSnapshot
	Flush() error
	StateHash() []byte
	ExtensionData() []byte
	BTPData() []byte
	Database() db.Database
}

// WorldState represents world state.
// You may change
type WorldState interface {
	GetAccountState(id []byte) AccountState
	GetAccountSnapshot(id []byte) AccountSnapshot
	GetSnapshot() WorldSnapshot
	GetValidatorState() ValidatorState
	GetExtensionState() ExtensionState
	GetBTPState() BTPState
	Reset(snapshot WorldSnapshot) error
	ClearCache()
	EnableNodeCache()
	NodeCacheEnabled() bool
	Database() db.Database
	EnableAccountNodeCache(id []byte) bool
}

type worldSnapshotImpl struct {
	database   db.Database
	accounts   trie.ImmutableForObject
	validators ValidatorSnapshot
	extension  ExtensionSnapshot
	btp        BTPSnapshot
}

func (ws *worldSnapshotImpl) GetValidatorSnapshot() ValidatorSnapshot {
	return ws.validators
}

func (ws *worldSnapshotImpl) GetExtensionSnapshot() ExtensionSnapshot {
	return ws.extension
}

func (ws *worldSnapshotImpl) ExtensionData() []byte {
	if ws.extension != nil {
		return ws.extension.Bytes()
	}
	return nil
}

func (ws *worldSnapshotImpl) GetBTPSnapshot() BTPSnapshot {
	return ws.btp
}

func (ws *worldSnapshotImpl) BTPData() []byte {
	if ws.btp != nil {
		return ws.btp.Bytes()
	}
	return nil
}

func (ws *worldSnapshotImpl) StateHash() []byte {
	return ws.accounts.Hash()
}

func (ws *worldSnapshotImpl) Flush() error {
	if ws.btp != nil {
		if err := ws.btp.Flush(); err != nil {
			return err
		}
	}
	if ass, ok := ws.accounts.(trie.SnapshotForObject); ok {
		if err := ass.Flush(); err != nil {
			return err
		}
	}
	if ws.extension != nil {
		if err := ws.extension.Flush(); err != nil {
			return err
		}
	}
	return ws.validators.Flush()
}

func (ws *worldSnapshotImpl) Database() db.Database {
	return ws.database
}

func (ws *worldSnapshotImpl) GetAccountSnapshot(id []byte) AccountSnapshot {
	key := addressIDToKey(id)
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Errorf("Fail to get account for %x err=%v", key, err)
		return nil
	}
	if obj == nil {
		return nil
	}
	if s, ok := obj.(*accountSnapshotImpl); ok {
		return s
	} else {
		log.Errorf("Returned account isn't accountSnapshotImpl type=%T", obj)
		return nil
	}
}

type worldStateImpl struct {
	mutex sync.Mutex

	database        db.Database
	accounts        trie.MutableForObject
	mutableAccounts map[string]AccountState
	lastAccounts    map[string]AccountSnapshot
	validators      ValidatorState
	extension       extensionStateHolder
	btp             BTPState

	nodeCacheEnabled bool
}

func (ws *worldStateImpl) GetValidatorState() ValidatorState {
	return ws.validators
}

func (ws *worldStateImpl) GetExtensionState() ExtensionState {
	return ws.extension.GetState()
}

func (ws *worldStateImpl) GetBTPState() BTPState {
	return ws.btp
}

func (ws *worldStateImpl) Reset(isnapshot WorldSnapshot) error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	snapshot := isnapshot.(*worldSnapshotImpl)
	if ws.database != snapshot.database {
		return errors.InvalidStateError.New("InvalidSnapshotWithDifferentDB")
	}
	ws.accounts.Reset(snapshot.accounts)
	for ids, as := range ws.mutableAccounts {
		key := as.(*accountStateImpl).key
		if value := ws.getAccountSnapshotWithKey(key); value == nil {
			as.Clear()
			delete(ws.lastAccounts, ids)
		} else {
			if err := as.Reset(value); err != nil {
				return err
			}
			ws.lastAccounts[ids] = value
		}
	}
	ws.validators.Reset(snapshot.GetValidatorSnapshot())
	ws.extension.Reset(snapshot.GetExtensionSnapshot())
	ws.btp.Reset(snapshot.GetBTPSnapshot())
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
	as := ws.getAccountSnapshotWithKey(key)
	ac := newAccountState(ws.database, as, key, ws.nodeCacheEnabled)
	ws.mutableAccounts[ids] = ac
	ws.lastAccounts[ids] = as
	return ac
}

func (ws *worldStateImpl) flushAccountCacheInLock() {
	for ids, as := range ws.mutableAccounts {
		key := as.(*accountStateImpl).key
		s := as.GetSnapshot()
		if ass, _ := ws.lastAccounts[ids]; ass != nil {
			if ass == s {
				continue
			}
		} else {
			if s.IsEmpty() {
				continue
			}
		}
		ws.lastAccounts[ids] = s
		if s.IsEmpty() {
			if _, err := ws.accounts.Delete(key); err != nil {
				log.Errorf("Fail to delete account key = %x, err=%+v", key, err)
			}
		} else {
			if _, err := ws.accounts.Set(key, s); err != nil {
				log.Errorf("Fail to set snapshot for %x, err=%+v", key, err)
			}
		}
	}
}

func (ws *worldStateImpl) ClearCache() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	ws.flushAccountCacheInLock()
	ws.accounts.ClearCache()
	ws.extension.ClearCache()
	ws.mutableAccounts = make(map[string]AccountState)
	ws.lastAccounts = make(map[string]AccountSnapshot)
}

func (ws *worldStateImpl) EnableNodeCache() {
	ws.nodeCacheEnabled = true
	if cache := cache.WorldNodeCacheOf(ws.database); cache != nil {
		trie_manager.SetCacheOfMutableForObject(ws.accounts, cache)
	}
}

func (ws *worldStateImpl) NodeCacheEnabled() bool {
	return ws.nodeCacheEnabled
}

func (ws *worldStateImpl) Database() db.Database {
	return ws.database
}

func (ws *worldStateImpl) EnableAccountNodeCache(id []byte) bool {
	if ws.nodeCacheEnabled {
		return cache.EnableAccountNodeCacheByForce(ws.database, addressIDToKey(id))
	}
	return false
}

func (ws *worldStateImpl) getAccountSnapshotWithKey(key []byte) AccountSnapshot {
	obj, err := ws.accounts.Get(key)
	if err != nil {
		log.Errorf("Fail to get account for %x err=%+v", key, err)
		return nil
	}
	if obj == nil {
		return nil
	} else {
		return obj.(AccountSnapshot)
	}
}

func (ws *worldStateImpl) GetAccountSnapshot(id []byte) AccountSnapshot {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if a, ok := ws.mutableAccounts[string(id)]; ok {
		return a.GetSnapshot()
	}

	key := addressIDToKey(id)
	if ass := ws.getAccountSnapshotWithKey(key); ass != nil {
		return ass
	} else {
		return newAccountSnapshot(ws.database)
	}
}

func (ws *worldStateImpl) GetSnapshot() WorldSnapshot {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	ws.flushAccountCacheInLock()

	return &worldSnapshotImpl{
		database:   ws.database,
		accounts:   ws.accounts.GetSnapshot(),
		validators: ws.validators.GetSnapshot(),
		extension:  ws.extension.GetSnapshot(),
		btp:        ws.btp.GetSnapshot(),
	}
}

func NewWorldState(
	database db.Database, stateHash []byte, vs ValidatorSnapshot, es ExtensionSnapshot, btpHash []byte,
) WorldState {
	ws := new(worldStateImpl)
	ws.database = database
	ws.accounts = trie_manager.NewMutableForObject(database, stateHash, AccountType)
	ws.mutableAccounts = make(map[string]AccountState)
	ws.lastAccounts = make(map[string]AccountSnapshot)
	if vs == nil {
		ws.validators, _ = ValidatorStateFromHash(database, nil)
	} else {
		ws.validators = ValidatorStateFromSnapshot(vs)
	}
	ws.extension.Reset(es)
	ws.btp = NewBTPState(database, btpHash)
	return ws
}

func NewWorldSnapshot(
	dbase db.Database, stateHash []byte, vs ValidatorSnapshot, es ExtensionSnapshot, btpData []byte,
) WorldSnapshot {
	ws := new(worldSnapshotImpl)
	ws.database = dbase
	ws.accounts = trie_manager.NewImmutableForObject(dbase, stateHash, AccountType)
	if vs == nil {
		vs, _ = ValidatorSnapshotFromHash(dbase, nil)
	}
	ws.validators = vs
	ws.extension = es
	ws.btp = NewBTPSnapshot(dbase, btpData)
	return ws
}

func NewWorldSnapshotWithNewValidators(dbase db.Database, snapshot WorldSnapshot, vss ValidatorSnapshot) WorldSnapshot {
	if ws, ok := snapshot.(*worldSnapshotImpl); ok {
		return &worldSnapshotImpl{
			database:   ws.database,
			accounts:   ws.accounts,
			validators: vss,
			extension:  ws.extension,
			btp:        ws.btp,
		}
	} else {
		return NewWorldSnapshot(dbase, snapshot.StateHash(), vss, snapshot.GetExtensionSnapshot(), snapshot.BTPData())
	}
}

func WorldStateFromSnapshot(wss WorldSnapshot) (WorldState, error) {
	if wss, ok := wss.(*worldSnapshotImpl); ok {
		ws := new(worldStateImpl)
		ws.database = wss.database
		ws.accounts = trie_manager.NewMutableFromImmutableForObject(wss.accounts)
		ws.mutableAccounts = make(map[string]AccountState)
		ws.lastAccounts = make(map[string]AccountSnapshot)
		ws.validators = ValidatorStateFromSnapshot(wss.GetValidatorSnapshot())
		ws.extension.Reset(wss.GetExtensionSnapshot())
		ws.btp = NewBTPState(wss.database, wss.BTPData())
		return ws, nil
	}
	return nil, errors.ErrIllegalArgument
}

func NewWorldSnapshotWithBuilder(
	builder merkle.Builder, sh []byte, vh []byte, ess ExtensionSnapshot, bh []byte,
) (WorldSnapshot, error) {
	ws := new(worldSnapshotImpl)
	ws.database = builder.Database()
	ws.accounts = trie_manager.NewImmutableForObject(ws.database, sh, AccountType)
	ws.accounts.Resolve(builder)
	if vs, err := NewValidatorSnapshotWithBuilder(builder, vh); err != nil {
		return nil, err
	} else {
		ws.validators = vs
	}
	ws.extension = ess
	ws.btp = NewBTPSnapshot(ws.database, bh)
	return ws, nil
}
