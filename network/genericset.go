package network

import "sync"

type GenericSet[T comparable] struct {
	m       map[T]interface{}
	version int64
	mtx     sync.RWMutex
}

func (s *GenericSet[T]) _add(v T, d interface{}) bool {
	if _, ok := s.m[v]; !ok {
		s.m[v] = d
		return true
	}
	return false
}

func (s *GenericSet[T]) _remove(v T) bool {
	if _, ok := s.m[v]; ok {
		delete(s.m, v)
		return true
	}
	return false
}

func (s *GenericSet[T]) _clear() bool {
	if len(s.m) > 0 {
		s.m = make(map[T]interface{})
		return true
	}
	return false
}

func (s *GenericSet[T]) _contains(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *GenericSet[T]) _array() []T {
	arr := make([]T, 0)
	for v := range s.m {
		arr = append(arr, v)
	}
	return arr
}

func (s *GenericSet[T]) _len() int {
	return len(s.m)
}

func (s *GenericSet[T]) Add(v T) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s._add(v, true) {
		s.version++
		return true
	}
	return false
}

func (s *GenericSet[T]) AddIf(v T, predicate func(v T) bool) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if predicate == nil || predicate(v) {
		return s._add(v, true)
	}
	return false
}

func (s *GenericSet[T]) Remove(v T) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s._remove(v) {
		s.version++
		return true
	}
	return false
}

func (s *GenericSet[T]) Clear() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s._clear() {
		s.version++
	}
}

func (s *GenericSet[T]) Reset(l ...T) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s._clear() {
		for _, v := range l {
			s._add(v, true)
		}
		s.version++
	}
}

func (s *GenericSet[T]) Contains(v T) bool {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s._contains(v)
}

func (s *GenericSet[T]) Find(predicate func(v T) bool) []T {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	var l []T
	for v := range s.m {
		if predicate == nil || predicate(v) {
			l = append(l, v)
		}
	}
	return l
}

func (s *GenericSet[T]) FindOne(predicate func(v T) bool) (T, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	for v := range s.m {
		if predicate == nil || predicate(v) {
			return v, true
		}
	}
	var v T
	return v, false
}

func (s *GenericSet[T]) Len() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s._len()
}

func (s *GenericSet[T]) IsEmpty() bool {
	return s.Len() == 0
}

func (s *GenericSet[T]) Array() []T {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s._array()
}

func (s *GenericSet[T]) Version() int64 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.version
}

func NewGenericSet[T comparable]() *GenericSet[T] {
	return &GenericSet[T]{
		m: make(map[T]interface{}),
	}
}
