package service

import (
	"github.com/pkg/errors"
	"log"
	"sync"
)

const (
	accountNoLock      = 0
	accountReadLock    = 1
	accountWriteLock   = 2
	accountWriteUnlock = 3
)

type lockRequest struct {
	id   string
	lock int
}

type lockedAccountState struct {
	lock   int
	state  accountState
	depend *worldVirtualState
}

type worldVirtualState struct {
	mutex  sync.Mutex
	waitor *sync.Cond

	parent    *worldVirtualState
	real      worldState
	base      worldSnapshot
	committed worldSnapshot

	accountStates map[string]*lockedAccountState
	worldLock     int
}

func (wvs *worldVirtualState) getAccountState(id []byte) accountState {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	las, ok := wvs.accountStates[string(id)]
	if ok {
		if las.depend != nil {
			las.depend.waitCommit()
			if las.lock == accountWriteLock {
				las.state = wvs.real.getAccountState(id)
			} else {
				las.state = las.depend.getAccountROState(id)
			}
			las.depend = nil
		}
		return las.state
	}

	if wvs.worldLock != accountNoLock {
		if wvs.base == nil {
			wvs.parent.realize()
			wvs.base = wvs.parent.committed
		}
		if wvs.worldLock == accountWriteLock {
			return wvs.real.getAccountState(id)
		} else {
			return newAccountROState(wvs.base.getAccountSnapshot(id))
		}
	}
	return nil
}

func (wvs *worldVirtualState) getAccountROState(id []byte) accountState {
	as := wvs.getAccountState(id)
	return newAccountROState(as.getSnapshot())
}

func (wvs *worldVirtualState) getSnapshot() worldSnapshot {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	wvss := new(worldVirtualSnapshot)
	wvss.parent = wvs
	switch wvs.worldLock {
	case accountWriteUnlock:
		wvss.base = wvs.committed
	case accountWriteLock:
		wvs.realizeBaseInLock()
		wvss.base = wvs.real.getSnapshot()
	case accountReadLock:
		wvs.realizeBaseInLock()
		fallthrough
	default:
		if wvs.committed != nil {
			wvss.base = wvs.committed
		} else {
			wvss.base = wvs.base
			wvss.accountSnapshots = map[string]accountSnapshot{}
			for id, las := range wvs.accountStates {
				if las.lock == accountWriteLock && las.depend == nil {
					wvss.accountSnapshots[id] = las.state.getSnapshot()
				}
			}
		}
	}
	return wvss
}

func (wvs *worldVirtualState) reset(snapshot worldSnapshot) error {
	s, ok := snapshot.(*worldVirtualSnapshot)
	if !ok {
		return errors.New("InvalidSnapshot")
	}
	if wvs != s.parent {
		return errors.New("InvalidSnapshot")
	}
	// TODO check implementation
	// if wvs.accounts != nil && s.real != nil {
	// 	if wvs.accounts.lock == accountWriteLock {
	// 		err := wvs.real.reset(s.real)
	// 		if err != nil {
	// 			log.Panic(err)
	// 		}
	// 	}
	// }
	for id, ass := range s.accountSnapshots {
		err := wvs.accountStates[id].state.reset(ass)
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
		if as.lock == accountWriteLock {
			wvs.mutex.Unlock()
			return wvs, true
		}
		depend := as.depend
		wvs.mutex.Unlock()
		return depend, true
	}
	switch wvs.worldLock {
	case accountWriteLock:
		return wvs, true
	case accountReadLock:
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

func (wvs *worldVirtualState) waitCommit() {
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

func (wvs *worldVirtualState) realize() {
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
			ws.committed = ws.real.getSnapshot()
		}
		ws.mutex.Unlock()
	}
}

func (wvs *worldVirtualState) realizeBaseInLock() {
	if wvs.base != nil {
		return
	}
	wvs.parent.realize()
	wvs.base = wvs.committed
}

func (wvs *worldVirtualState) getFuture(reqs []lockRequest) *worldVirtualState {
	nwvs := new(worldVirtualState)
	nwvs.real = wvs.real
	nwvs.waitor = sync.NewCond(&nwvs.mutex)
	nwvs.base = wvs.committed
	nwvs.parent = wvs
	applyLockRequests(nwvs, reqs)

	return nwvs
}

func applyLockRequests(wvs *worldVirtualState, reqs []lockRequest) {
	wvs.accountStates = make(map[string]*lockedAccountState, len(reqs))
	for _, req := range reqs {
		if req.id == "" {
			wvs.worldLock = req.lock
		} else {
			var depend *worldVirtualState
			if wvs.parent != nil {
				depend = wvs.parent.getDepend(req.id)
			}

			las := new(lockedAccountState)
			las.lock = req.lock
			if depend != nil {
				las.depend = depend
			} else {
				las.state = wvs.real.getAccountState([]byte(req.id))
			}
			wvs.accountStates[req.id] = las
		}
	}
}

func (wvs *worldVirtualState) commit() {
	wvs.mutex.Lock()
	defer wvs.mutex.Unlock()

	if wvs.waitor == nil {
		return
	}

	for _, las := range wvs.accountStates {
		if las.lock == accountWriteLock {
			las.lock = accountWriteUnlock
			ass := las.state.getSnapshot()
			las.state = newAccountROState(ass)
		}
	}

	if wvs.worldLock == accountWriteLock {
		wvs.worldLock = accountWriteUnlock
		wvs.committed = wvs.real.getSnapshot()
	}

	wvs.waitor.Broadcast()
	wvs.waitor = nil
	return
}

func newWorldVirtualState(ws worldState, reqs []lockRequest) *worldVirtualState {
	nwvs := new(worldVirtualState)
	nwvs.real = ws
	nwvs.base = ws.getSnapshot()
	if len(reqs) == 0 {
		nwvs.committed = nwvs.base
	} else {
		nwvs.waitor = sync.NewCond(&nwvs.mutex)
		applyLockRequests(nwvs, reqs)
	}
	return nwvs
}

type worldVirtualSnapshot struct {
	parent           *worldVirtualState
	base             worldSnapshot
	accountSnapshots map[string]accountSnapshot
}

func (wvss *worldVirtualSnapshot) getAccountSnapshot(id []byte) accountSnapshot {
	if len(wvss.accountSnapshots) > 0 {
		ass, ok := wvss.accountSnapshots[string(id)]
		if ok {
			return ass
		}
	}
	if wvss.base != nil {
		return wvss.base.getAccountSnapshot(id)
	}
	return nil
}

func (wvss *worldVirtualSnapshot) realize() {
	// TODO Implement realize
}

func (wvss *worldVirtualSnapshot) flush() error {
	// TODO realize itself, then flush
	if wvss.base != nil && len(wvss.accountSnapshots) == 0 {
		return wvss.base.flush()
	}
	return errors.New("NotAllowed")
}

func (wvss *worldVirtualSnapshot) stateHash() []byte {
	// TODO realize itself, then get stateHash
	if wvss.base != nil && len(wvss.accountSnapshots) == 0 {
		return wvss.base.stateHash()
	}
	return nil
}
