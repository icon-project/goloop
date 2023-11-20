package network

import (
	"bytes"
	"sort"
	"sync"

	"github.com/icon-project/goloop/module"
)

type ValidatorSet struct {
	l      []module.PeerID
	lHash  []byte
	sHash  []byte
	height int64
}

//TODO temporary
func (s *ValidatorSet) Get(i int) module.PeerID {
	if s == nil || i < 0 || len(s.l) <= i {
		return nil
	}
	return s.l[i]
}

func (s *ValidatorSet) ContainsTwoThird(predicate func(id module.PeerID) bool) bool {
	if s == nil {
		return false
	}
	n := 0
	for _, v := range s.l {
		if predicate(v) {
			n++
		}
	}
	return (len(s.l) * 2) < (n * 3)
}

func (s *ValidatorSet) Hash() []byte {
	if s == nil {
		return nil
	}
	return s.sHash
}

func (s *ValidatorSet) LHash() []byte {
	if s == nil {
		return nil
	}
	return s.lHash
}

func (s *ValidatorSet) Height() int64 {
	if s == nil {
		return 0
	}
	return s.height
}

func (s *ValidatorSet) Equal(v *ValidatorSet) bool {
	if s == nil {
		return v == nil
	}
	if bytes.Equal(s.LHash(), v.LHash()) {
		return true
	}
	if bytes.Equal(s.Hash(), v.Hash()) {
		return true
	}
	return false
}

func (s *ValidatorSet) Contains(id module.PeerID) bool {
	if s == nil {
		return false
	}
	for _, v := range s.l {
		if v.Equal(id) {
			return true
		}
	}
	return false
}

func NewValidatorSet(blk module.Block) *ValidatorSet {
	vl := blk.NextValidators()
	s := &ValidatorSet{
		l:      nil,
		lHash:  blk.NextValidatorsHash(),
		sHash:  nil,
		height: blk.Height(),
	}
	lLen := vl.Len()
	for i := 0; i < lLen; i++ {
		v, _ := vl.Get(i)
		s.l = append(s.l, NewPeerIDFromAddress(v.Address()))
	}
	sort.Slice(s.l, func(i, j int) bool {
		return bytes.Compare(s.l[i].Bytes(), s.l[j].Bytes()) < 0
	})
	var b []byte
	for _, id := range s.l {
		b = append(b, id.Bytes()...)
	}
	return s
}

type ValidatorSetCache struct {
	l     []*ValidatorSet
	size  int
	len   int
	write int
	mtx   sync.RWMutex
}

func (c *ValidatorSetCache) _add(v *ValidatorSet) {
	if c.size > c.len {
		c.len++
	}
	c.l[c.write] = v
	c.write++
	if c.write == c.size {
		c.write = 0
	}
}

func (c *ValidatorSetCache) _get(lHash []byte) *ValidatorSet {
	for _, v := range c.l {
		if bytes.Equal(v.LHash(), lHash) {
			return v
		}
	}
	return nil
}

func (c *ValidatorSetCache) _last() *ValidatorSet {
	if c.write == 0 {
		return nil
	}
	idx := c.write - 1
	if idx < 0 {
		idx = c.size - 1
	}
	return c.l[idx]
}

func (c *ValidatorSetCache) _first() *ValidatorSet {
	if c.write == 0 {
		return nil
	}
	idx := c.write + 1
	if c.size > c.len || idx == c.size {
		idx = 0
	}
	return c.l[idx]
}

func (c *ValidatorSetCache) Update(v *ValidatorSet) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c._add(v)
	return c._last().Equal(v)
}

func (c *ValidatorSetCache) Last() *ValidatorSet {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c._last()
}

func (c *ValidatorSetCache) Get(lHash []byte) *ValidatorSet {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c._get(lHash)
}

func (c *ValidatorSetCache) Height() int64 {
	return c.Last().Height()
}

func NewValidatorSetCache(size int) *ValidatorSetCache {
	return &ValidatorSetCache{
		l:    make([]*ValidatorSet, size),
		size: size,
	}
}
