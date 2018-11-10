package service

import (
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
		}
		return las.state
	}

	if wvs.worldLock != AccountNoLock {
		if wvs.base == nil {
			wvs.parent.Realize()
			wvs.base = wvs.parent.committed
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
	switch wvs.worldLock {
	case AccountWriteUnlock:
		wvss.base = wvs.committed
	case AccountWriteLock:
		wvs.realizeBaseInLock()
		wvss.base = wvs.real.GetSnapshot()
	case AccountReadLock:
		wvs.realizeBaseInLock()
		fallthrough
	default:
		if wvs.committed != nil {
			wvss.base = wvs.committed
		} else {
			wvss.base = wvs.base
			wvss.accountSnapshots = map[string]AccountSnapshot{}
			for id, las := range wvs.accountStates {
				if las.lock == AccountWriteLock && las.depend == nil {
					wvss.accountSnapshots[id] = las.state.GetSnapshot()
				}
			}
		}
	}
	return wvss
}

func (wvs *worldVirtualState) Reset(snapshot WorldSnapshot) error {
	s, ok := snapshot.(*worldVirtualSnapshot)
	if !ok {
		return errors.New("InvalidSnapshot")
	}
	if wvs != s.origin {
		return errors.New("InvalidSnapshot")
	}

	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.waitor != nil {
		return errors.New("AlreadyCommitted")
	}

	for id, ass := range s.accountSnapshots {
		err := wvs.accountStates[id].state.Reset(ass)
		if err != nil {
			log.Panic(err)
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
	wvs.accountStates = make(map[string]*lockedAccountState, len(reqs))
	for _, req := range reqs {
		if req.ID == "" {
			wvs.worldLock = req.Lock
		} else {
			var depend *worldVirtualState
			if wvs.parent != nil {
				depend = wvs.parent.getDepend(req.ID)
			}

			las := new(lockedAccountState)
			las.lock = req.Lock
			if depend != nil {
				las.depend = depend
			} else {
				las.state = wvs.real.GetAccountState([]byte(req.ID))
			}
			wvs.accountStates[req.ID] = las
		}
	}
}

func (wvs *worldVirtualState) Commit() {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.waitor == nil {
		return
	}

	for _, las := range wvs.accountStates {
		if las.lock == AccountWriteLock {
			las.lock = AccountWriteUnlock
			ass := las.state.GetSnapshot()
			las.state = newAccountROState(ass)
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

func (wvss *worldVirtualSnapshot) Realize() {
	// TODO Implement realize
}

func (wvss *worldVirtualSnapshot) Flush() error {
	// TODO realize itself, then flush
	if wvss.base != nil && len(wvss.accountSnapshots) == 0 {
		return wvss.base.Flush()
	}
	return errors.New("NotAllowed")
}

func (wvss *worldVirtualSnapshot) StateHash() []byte {
	// TODO realize itself, then get stateHash
	if wvss.base != nil && len(wvss.accountSnapshots) == 0 {
		return wvss.base.StateHash()
	}
	return nil
}
