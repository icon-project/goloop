package network

type DiffType int

const (
	DiffTypeNone DiffType = iota
	DiffTypePartial
	DiffTypeEntire
)

type Diff[T comparable] struct {
	Type DiffType
	Add  *GenericSet[T]
	Del  *GenericSet[T]
}

type DiffResult[T comparable] struct {
	From    int64
	Version int64
	Type    DiffType
	Add     []T
	Del     []T
}

type DiffSet[T comparable] struct {
	*GenericSet[T]
	version int64
	minVer  int64
	maxHold int64
	ds      map[int64]*Diff[T]
}

func (s *DiffSet[T]) _putDiff(dt DiffType) *Diff[T] {
	d := &Diff[T]{
		dt,
		NewGenericSet[T](),
		NewGenericSet[T](),
	}
	s.ds[s.version] = d
	s.version++
	if (s.version - s.minVer) > s.maxHold {
		delete(s.ds, s.minVer)
		s.minVer++
	}
	return d
}

func (s *DiffSet[T]) _add(v T) {
	for _, d := range s.ds {
		if !d.Del.Remove(v) {
			d.Add.Add(v)
		}
	}
}

func (s *DiffSet[T]) _remove(v T) {
	for _, d := range s.ds {
		if !d.Add.Remove(v) {
			d.Del.Add(v)
		}
	}
}

func (s *DiffSet[T]) _clear() {
	for _, d := range s.ds {
		d.Type = DiffTypeEntire
		d.Add.Clear()
		d.Del.Clear()
	}
}

func (s *DiffSet[T]) Version() int64 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.version
}

func (s *DiffSet[T]) MinVersion() int64 {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.minVer
}

func (s *DiffSet[T]) Diff(version int64) *DiffResult[T] {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	if s.version < version {
		return nil
	}
	if s.minVer > version {
		return &DiffResult[T]{
			From:    version,
			Version: s.version,
			Type:    DiffTypeEntire,
			Add:     s.GenericSet.Array(),
		}
	}
	if s.version == version {
		return &DiffResult[T]{
			From:    version,
			Version: s.version,
			Type:    DiffTypeNone,
		}
	}
	d := s.ds[version]
	return &DiffResult[T]{
		From:    version,
		Version: s.version,
		Type:    d.Type,
		Add:     d.Add.Array(),
		Del:     d.Del.Array(),
	}
}

func (s *DiffSet[T]) Add(v T) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.GenericSet._add(v, true) {
		return false
	}
	s._add(v)
	s._putDiff(DiffTypePartial).Add.Add(v)
	return true
}

func (s *DiffSet[T]) Remove(v T) bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.GenericSet._remove(v) {
		return false
	}
	s._remove(v)
	s._putDiff(DiffTypePartial).Del.Add(v)
	return true
}

func (s *DiffSet[T]) Clear() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.GenericSet._clear() {
		return
	}
	s._clear()
	s._putDiff(DiffTypeEntire)
}

func NewDiffSet[T comparable](size int) *DiffSet[T] {
	return &DiffSet[T]{
		GenericSet: NewGenericSet[T](),
		version:    0,
		minVer:     0,
		maxHold:    int64(size),
		ds:         make(map[int64]*Diff[T]),
	}
}
