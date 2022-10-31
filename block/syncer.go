package block

import "sync"

type syncer struct {
	mutex sync.Mutex
	lcbs  []func()
	cbs   []func()
}

func (s *syncer) begin() {
	s.mutex.Lock()
}

func (s *syncer) callLater(cb func()) {
	s.cbs = append(s.cbs, cb)
}

func (s *syncer) callLaterInLock(cb func()) {
	s.lcbs = append(s.lcbs, cb)
}

func (s *syncer) end() {
	for s.lcbs != nil {
		lcbs := s.lcbs
		s.lcbs = nil
		for _, cb := range lcbs {
			cb()
		}
	}
	cbs := s.cbs
	s.cbs = nil
	s.mutex.Unlock()
	for _, cb := range cbs {
		cb()
	}
}

func (s *syncer) Lock() {
	s.begin()
}

func (s *syncer) Unlock() {
	s.end()
}
