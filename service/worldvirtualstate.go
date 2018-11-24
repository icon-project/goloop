package service

import (
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
	"log"
	"sync"
)

const (
	AccountNoLock      = 0
	AccountReadLock    = 1
	AccountWriteLock   = 2
	AccountWriteUnlock = 3
)

type LockRequest struct {
	ID   string
	Lock int
}

type WorldVirtualState interface {
	WorldState
	GetFuture(reqs []LockRequest) WorldVirtualState
	WaitCommit()
	Commit()
	Realize()
}

type lockedAccountState struct {
	lock   int
	state  AccountState
	base   AccountSnapshot
	depend *worldVirtualState
}

type worldVirtualState struct {
	mutex  sync.Mutex
	waitor *sync.Cond

	parent    *worldVirtualState
	real      WorldState
	base      WorldSnapshot
	committed WorldSnapshot

	accountStates map[string]*lockedAccountState
	worldLock     int
}

func (wvs *worldVirtualState) GetValidators() module.ValidatorList {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.worldLock != AccountNoLock {
		wvs.realizeBaseInLock()

		if wvs.committed != nil {
			return wvs.committed.GetValidators()
		}
		if wvs.worldLock == AccountWriteLock {
			return wvs.real.GetValidators()
		} else {
			return wvs.base.GetValidators()
		}
	}
	return nil
}

func (wvs *worldVirtualState) SetValidators(vl []module.Validator) error {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.worldLock == AccountWriteLock {
		return wvs.real.SetValidators(vl)
	}
	return errors.New("ValidatorListIsNotLocked")
}

func (wvs *worldVirtualState) GetAccountSnapshot(id []byte) AccountSnapshot {
	as := wvs.GetAccountState(id)
	if as == nil {
		return nil
	}
	return as.GetSnapshot()
}

func (wvs *worldVirtualState) GetAccountState(id []byte) AccountState {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	las, ok := wvs.accountStates[string(id)]
	if ok {
		if las.depend != nil {
			las.depend.WaitCommit()
			if las.lock == AccountWriteLock {
				las.state = wvs.real.GetAccountState(id)
			} else {
				las.state = las.depend.GetAccountROState(id)
			}
			las.depend = nil
			las.base = las.state.GetSnapshot()
		}
		return las.state
	}

	if wvs.worldLock != AccountNoLock {
		wvs.realizeBaseInLock()
		if wvs.committed != nil {
			return newAccountROState(wvs.committed.GetAccountSnapshot(id))
		}
		if wvs.worldLock == AccountWriteLock {
			return wvs.real.GetAccountState(id)
		} else {
			return newAccountROState(wvs.base.GetAccountSnapshot(id))
		}
	}
	return nil
}

func (wvs *worldVirtualState) GetAccountROState(id []byte) AccountState {
	as := wvs.GetAccountState(id)
	return newAccountROState(as.GetSnapshot())
}

func (wvs *worldVirtualState) GetSnapshot() WorldSnapshot {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	wvss := new(worldVirtualSnapshot)
	wvss.origin = wvs

	// If we have final snapshot, we can use it.
	if wvs.committed != nil {
		wvss.base = wvs.committed
		return wvss
	}

	switch wvs.worldLock {
	case AccountWriteUnlock:
		panic("Not possible to get here.")
	case AccountWriteLock:
		wvs.realizeBaseInLock()
		wvss.base = wvs.real.GetSnapshot()
	case AccountReadLock:
		wvs.realizeBaseInLock()
		fallthrough
	default:
		wvss.base = wvs.base
		wvss.accountSnapshots = map[string]AccountSnapshot{}
		for id, las := range wvs.accountStates {
			if las.lock == AccountWriteLock && las.state != nil {
				wvss.accountSnapshots[id] = las.state.GetSnapshot()
			}
		}
	}
	return wvss
}

func (wvs *worldVirtualState) Reset(snapshot WorldSnapshot) error {
	wvss, ok := snapshot.(*worldVirtualSnapshot)
	if !ok {
		return errors.New("InvalidSnapshot")
	}
	if wvs != wvss.origin {
		return errors.New("InvalidSnapshot")
	}

	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.waitor == nil {
		return errors.New("AlreadyCommitted")
	}

	if wvs.worldLock == AccountWriteLock {
		return wvs.real.Reset(wvss.base)
	}
	for id, las := range wvs.accountStates {
		if las.lock != AccountWriteLock {
			continue
		}
		var err error
		if ass, ok := wvss.accountSnapshots[id]; ok {
			err = las.state.Reset(ass)
		} else {
			// when it makes snapshot, it may not be realized.
			// but it may have been changed, so we need to recover it.
			if las.state != nil {
				if wvss.base != nil {
					err = las.state.Reset(wvss.base.GetAccountSnapshot([]byte(id)))
				} else {
					err = las.state.Reset(las.base)
				}
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (wvs *worldVirtualState) checkDepend(id string) (*worldVirtualState, bool) {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	// it's already success to make result
	// nobody need to wait to be finished
	if wvs.committed != nil {
		return nil, true
	}
	if as, ok := wvs.accountStates[id]; ok {
		if as.lock == AccountWriteLock {
			return wvs, true
		}
		return as.depend, true
	}
	switch wvs.worldLock {
	case AccountWriteLock:
		return wvs, true
	case AccountReadLock:
		if wvs.base != nil {
			return nil, true
		}
	}
	return nil, false
}

func (wvs *worldVirtualState) getDepend(id string) *worldVirtualState {
	for ws := wvs; ws != nil; ws = ws.parent {
		if dep, found := ws.checkDepend(id); found {
			return dep
		}
	}
	return nil
}

func (wvs *worldVirtualState) WaitCommit() {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	wvs.waitCommitInLock()
}

func (wvs *worldVirtualState) waitCommitInLock() {
	if wvs.waitor != nil {
		wvs.waitor.Wait()
		wvs.waitor = nil
	}
}

func (wvs *worldVirtualState) Realize() {
	wsList := make([]*worldVirtualState, 0)
	for ws := wvs; ws != nil; ws = ws.parent {
		ws.mutex.Lock()
		if ws.committed != nil {
			ws.mutex.Unlock()
			break
		}

		wsList = append(wsList, ws)

		if ws.base != nil {
			break
		}
	}

	for idx := len(wsList) - 1; idx >= 0; idx-- {
		ws := wsList[idx]
		ws.waitCommitInLock()
		if idx == 0 && ws.committed == nil {
			ws.committed = ws.real.GetSnapshot()
		}
		ws.mutex.Unlock()
	}
}

func (wvs *worldVirtualState) realizeBaseInLock() {
	if wvs.base != nil {
		return
	}
	wvs.parent.Realize()
	wvs.base = wvs.committed
}

func (wvs *worldVirtualState) getRealizedBase() WorldSnapshot {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	wvs.realizeBaseInLock()
	return wvs.base
}

func (wvs *worldVirtualState) GetFuture(reqs []LockRequest) WorldVirtualState {
	nwvs := new(worldVirtualState)
	nwvs.real = wvs.real
	nwvs.waitor = sync.NewCond(&nwvs.mutex)
	nwvs.base = wvs.committed
	nwvs.parent = wvs
	applyLockRequests(nwvs, reqs)

	return nwvs
}

func applyLockRequests(wvs *worldVirtualState, reqs []LockRequest) {
	for _, req := range reqs {
		if req.ID != "" {
			continue
		}
		switch req.Lock {
		case AccountWriteLock:
			wvs.worldLock = AccountWriteLock
			return
		case AccountReadLock:
			wvs.worldLock = AccountReadLock
		default:
			log.Panicf("World lock request with invalid value=%d", req.Lock)
		}
	}

	wvs.accountStates = make(map[string]*lockedAccountState, len(reqs))
	for _, req := range reqs {
		if req.ID == "" {
			continue
		}
		if req.Lock != AccountReadLock && req.Lock != AccountWriteLock {
			log.Panicf("Account(%x) invalid lock request req=%d",
				req.ID, req.Lock)
			continue
		}

		if las, ok := wvs.accountStates[req.ID]; ok {
			if las.lock != req.Lock && req.Lock == AccountWriteLock {
				las.lock = req.Lock
			}
		} else {
			wvs.accountStates[req.ID] = &lockedAccountState{lock: req.Lock}
		}
	}

	for id, las := range wvs.accountStates {
		if wvs.parent != nil {
			las.depend = wvs.parent.getDepend(id)
		} else {
			las.depend = nil
		}
		if las.depend == nil {
			idBytes := []byte(id)
			if las.lock == AccountWriteLock {
				las.state = wvs.real.GetAccountState(idBytes)
			} else {
				las.state = newAccountROState(
					wvs.real.GetAccountSnapshot(idBytes))
			}
		}
	}
}

func (wvs *worldVirtualState) Commit() {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.waitor == nil {
		return
	}

	for id, las := range wvs.accountStates {
		if las.lock == AccountWriteLock {
			las.lock = AccountWriteUnlock
			if las.depend != nil {
				las.depend.WaitCommit()
				las.state = las.depend.GetAccountROState([]byte(id))
				las.base = las.state.GetSnapshot()
				las.depend = nil
			} else {
				ass := las.state.GetSnapshot()
				las.state = newAccountROState(ass)
			}
		}
	}

	if wvs.worldLock == AccountWriteLock {
		wvs.worldLock = AccountWriteUnlock
		wvs.committed = wvs.real.GetSnapshot()
	}

	wvs.waitor.Broadcast()
	wvs.waitor = nil
	return
}

func NewWorldVirtualState(ws WorldState, reqs []LockRequest) WorldVirtualState {
	nwvs := new(worldVirtualState)
	nwvs.real = ws
	nwvs.base = ws.GetSnapshot()
	if len(reqs) == 0 {
		nwvs.committed = nwvs.base
	} else {
		nwvs.waitor = sync.NewCond(&nwvs.mutex)
		applyLockRequests(nwvs, reqs)
	}
	return nwvs
}

type worldVirtualSnapshot struct {
	origin           *worldVirtualState
	base             WorldSnapshot
	accountSnapshots map[string]AccountSnapshot
}

func (wvss *worldVirtualSnapshot) GetAccountSnapshot(id []byte) AccountSnapshot {
	if len(wvss.accountSnapshots) > 0 {
		ass, ok := wvss.accountSnapshots[string(id)]
		if ok {
			return ass
		}
	}
	if wvss.base != nil {
		return wvss.base.GetAccountSnapshot(id)
	}
	return nil
}

func (wvss *worldVirtualSnapshot) realize() error {
	if wvss.base == nil {
		wvss.base = wvss.origin.getRealizedBase()
	}
	if len(wvss.accountSnapshots) > 0 {
		ws, err := WorldStateFromSnapshot(wvss.base)
		if err != nil {
			return err
		}
		for id, ass := range wvss.accountSnapshots {
			as := ws.GetAccountState([]byte(id))
			as.Reset(ass)
		}
		wvss.accountSnapshots = nil
		wvss.base = ws.GetSnapshot()
	}
	return nil
}

func (wvss *worldVirtualSnapshot) Flush() error {
	if err := wvss.realize(); err != nil {
		return err
	}
	return wvss.base.Flush()
}

func (wvss *worldVirtualSnapshot) StateHash() []byte {
	if err := wvss.realize(); err != nil {
		return nil
	}
	return wvss.base.StateHash()
}

func (wvss *worldVirtualSnapshot) GetValidators() module.ValidatorList {
	if wvss.base != nil {
		return wvss.base.GetValidators()
	}
	return nil
}
