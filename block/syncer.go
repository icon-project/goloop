package block

import "sync"

type syncer struct {
	mutex sync.Mutex
	cbs   []func()
}

func (s *syncer) begin() {
	s.mutex.Lock()
}

func (s *syncer) callLater(cb func()) {
	s.cbs = append(s.cbs, cb)
}

func (s *syncer) end() {
	cbs := s.cbs
	s.cbs = nil
	s.mutex.Unlock()
	for _, cb := range cbs {
		cb()
	}
}
