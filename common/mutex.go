package common

import "sync"

// Mutex is wrapper for sync.Mutex.
type Mutex struct {
	mutex sync.Mutex
	bcbs  []func()
	acbs  []func()
}

// Lock acquires lock.
func (m *Mutex) Lock() {
	m.mutex.Lock()
}

// Unlock calls scheduled functions and releases lock.
func (m *Mutex) Unlock() {
	for i := 0; i < len(m.bcbs); i++ {
		m.bcbs[i]()
	}
	m.bcbs = nil
	acbs := m.acbs
	m.acbs = nil
	m.mutex.Unlock()
	for _, cb := range acbs {
		cb()
	}
}

// CallBeforeUnlock schedules to call f before next unlock. This function shall
// be called while lock is acquired.
func (m *Mutex) CallBeforeUnlock(f func()) {
	m.bcbs = append(m.bcbs, f)
}

// CallAfterUnlock schedules to call f after next unlock. This function shall be
// called while lock is acquired.
func (m *Mutex) CallAfterUnlock(f func()) {
	m.acbs = append(m.acbs, f)
}
