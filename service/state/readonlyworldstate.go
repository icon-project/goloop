package state

import (
	"sync"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type readOnlyWorldState struct {
	WorldSnapshot

	lock           sync.Mutex
	accounts       map[string]AccountState
	validatorState ValidatorState
	extensionState ExtensionState
	btp            BTPState
}

func (ws *readOnlyWorldState) GetExtensionState() ExtensionState {
	return ws.extensionState
}

func (ws *readOnlyWorldState) GetBTPState() BTPState {
	return ws.btp
}

func (ws *readOnlyWorldState) GetAccountState(id []byte) AccountState {
	ws.lock.Lock()
	defer ws.lock.Unlock()

	ids := string(id)
	if as, ok := ws.accounts[ids]; ok {
		return as
	}

	as := newAccountROState(ws.Database(), ws.WorldSnapshot.GetAccountSnapshot(id))
	ws.accounts[ids] = as

	return as
}

func (ws *readOnlyWorldState) GetSnapshot() WorldSnapshot {
	return ws.WorldSnapshot
}

func (ws *readOnlyWorldState) GetValidatorState() ValidatorState {
	return ws.validatorState
}

func (ws *readOnlyWorldState) Reset(snapshot WorldSnapshot) error {
	if ws.WorldSnapshot != snapshot {
		return errors.InvalidStateError.New(
			"readOnlyWorldState.Reset() with different snapshot")
	}
	return nil
}

func (ws *readOnlyWorldState) ClearCache() {
	// nothing to do
}

func (ws *readOnlyWorldState) EnableNodeCache() {
	// nothing to do
}

func (ws *readOnlyWorldState) NodeCacheEnabled() bool {
	return false
}

func (ws *readOnlyWorldState) EnableAccountNodeCache(id []byte) bool {
	return false
}

type readonlyValidatorState struct {
	ValidatorSnapshot
}

func (vs *readonlyValidatorState) Set([]module.Validator) error {
	return errors.InvalidStateError.New("ReadOnlyState")
}

func (vs *readonlyValidatorState) Add(v module.Validator) error {
	return errors.InvalidStateError.New("ReadOnlyState")
}

func (vs *readonlyValidatorState) Remove(v module.Validator) bool {
	return false
}

func (vs *readonlyValidatorState) GetSnapshot() ValidatorSnapshot {
	return vs.ValidatorSnapshot
}

func (vs *readonlyValidatorState) Reset(vss ValidatorSnapshot) {
	// do nothing
}

func newReadOnlyValidatorState(vss ValidatorSnapshot) ValidatorState {
	return &readonlyValidatorState{vss}
}

func newReadOnlyExtensionState(ess ExtensionSnapshot) ExtensionState {
	if ess == nil {
		return nil
	} else {
		return ess.NewState(true)
	}
}

func newReadOnlyBTPState(bss BTPSnapshot) BTPState {
	if bss == nil {
		return nil
	} else {
		return bss.NewState()
	}
}

func NewReadOnlyWorldState(wss WorldSnapshot) WorldState {
	return &readOnlyWorldState{
		WorldSnapshot:  wss,
		accounts:       make(map[string]AccountState),
		validatorState: newReadOnlyValidatorState(wss.GetValidatorSnapshot()),
		extensionState: newReadOnlyExtensionState(wss.GetExtensionSnapshot()),
		btp:            newReadOnlyBTPState(wss.GetBTPSnapshot()),
	}
}
