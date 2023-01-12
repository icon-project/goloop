package service

import (
	"container/list"
	"time"

	"github.com/icon-project/goloop/common/log"
)

type TXIDCache interface {
	Contains(id []byte, ts int64) bool
	Add(id []byte, ts int64)
	RemoveOldTXsByTS(ts int64)
	Len() int
}

type txIDSet map[string]struct{}

type txIDSlot struct {
	id    int64
	txIds txIDSet
}

func (s *txIDSlot) contains(txId []byte) bool {
	_, ok := s.txIds[string(txId)]
	return ok
}

func (s *txIDSlot) put(txId []byte) {
	s.txIds[string(txId)] = struct{}{}
}

func (s *txIDSlot) remove(txId []byte) {
	delete(s.txIds, string(txId))
}

func (s *txIDSlot) len() int {
	return len(s.txIds)
}

func (s *txIDSlot) flush() {
	s.txIds = make(txIDSet)
}

func newTxIDSlot(slotId int64) *txIDSlot {
	return &txIDSlot{id: slotId, txIds: make(txIDSet)}
}

type txIDCache struct {
	slotDuration int64
	slotSize     int
	slotMap      map[int64]*list.Element
	slotList     *list.List
	log          log.Logger
	txCount      int
	minTs        int64
}

func (c *txIDCache) getSlotId(ts int64) int64 {
	return ts / c.slotDuration
}

func (c *txIDCache) getSlot(slotId int64, createIfMissing bool) *txIDSlot {
	if e := c.slotMap[slotId]; e != nil {
		return e.Value.(*txIDSlot)
	}
	if createIfMissing {
		return c.newSlot(slotId)
	}
	return nil
}

func (c *txIDCache) newSlot(slotId int64) *txIDSlot {
	var e *list.Element

	for e = c.slotList.Back(); e != nil; e = e.Prev() {
		s := e.Value.(*txIDSlot)
		if s.id == slotId {
			c.log.Warnf("State inconsistency between slotList and slotMap: slotId=%d", slotId)
			return s
		}
		if s.id < slotId {
			break
		}
	}

	s := newTxIDSlot(slotId)
	if e == nil {
		e = c.slotList.PushFront(s)
	} else {
		e = c.slotList.InsertAfter(s, e)
	}

	c.slotMap[slotId] = e
	return s
}

func (c *txIDCache) verifyTx(id []byte, ts int64) bool {
	return id != nil && ts >= c.minTs
}

func (c *txIDCache) Contains(id []byte, ts int64) bool {
	if !c.verifyTx(id, ts) {
		return false
	}

	c.log.Tracef("Contains() start: id=%#x ts=%d", id, ts)

	var ok bool
	slotId := c.getSlotId(ts)
	if s := c.getSlot(slotId, false); s != nil {
		ok = s.contains(id)
	}

	c.log.Tracef("Contains() end: id=%#x ts=%d ret=%t", id, ts, ok)
	return ok
}

func (c *txIDCache) Add(id []byte, ts int64) {
	if !c.verifyTx(id, ts) {
		return
	}

	c.log.Tracef("Add() start: id=%#x ts=%d txCount=%d", id, ts, c.txCount)

	slotId := c.getSlotId(ts)
	s := c.getSlot(slotId, true)
	if s == nil {
		c.log.Warnf("Failed to create a txIDSlot: slotId=%d", slotId)
		return
	}

	if !s.contains(id) {
		c.flushSlotIfFull(slotId, s)
		s.put(id)
		c.txCount++
	} else {
		c.log.Infof("Already dropped TX: id=%#x ts=%d", id, ts)
	}
	c.log.Tracef("Add() end: id=%#x ts=%d txCount=%d", id, ts, c.txCount)
}

func (c *txIDCache) flushSlotIfFull(slotId int64, s *txIDSlot) {
	n := s.len()
	if n < c.slotSize {
		return
	}

	c.log.Warnf("Flush a full txIDSlot: slotId=%d slotSize=%d", slotId, c.slotSize)
	s.flush()
	c.txCount -= n
}

func (c *txIDCache) Len() int {
	return c.txCount
}

func (c *txIDCache) RemoveOldTXsByTS(ts int64) {
	if ts <= c.minTs {
		return
	}

	c.log.Tracef("RemoveOldTXsByTS() start: ts=%d txCount=%d slots=%d,%d",
		ts, c.txCount, c.slotList.Len(), len(c.slotMap))

	slotId := c.getSlotId(ts)
	for e := c.slotList.Front(); e != nil; {
		s := e.Value.(*txIDSlot)
		if s.id >= slotId {
			break
		}

		old := e
		e = e.Next()

		c.slotList.Remove(old)
		delete(c.slotMap, s.id)
		c.txCount -= s.len()

		c.log.Debugf(
			"Remove a txIDSlot: slotId=%d ts=%d txCount=%d slots=%d,%d",
			slotId, ts, c.txCount, c.slotList.Len(), len(c.slotMap))
	}

	c.minTs = ts
	c.log.Tracef("RemoveOldTXsByTS() end: ts=%d txCount=%d slots=%d,%d",
		ts, c.txCount, c.slotList.Len(), len(c.slotMap))
}

func NewTxIDCache(slotDuration time.Duration, slotSize int, logger log.Logger) TXIDCache {
	if logger == nil {
		logger = log.GlobalLogger()
	}
	logger.Infof("NewTxIDCache: slotDuration=%d slotSize=%d", slotDuration, slotSize)
	return &txIDCache{
		slotDuration: slotDuration.Microseconds(),
		slotSize:     slotSize,
		slotMap:      make(map[int64]*list.Element),
		slotList:     list.New(),
		log:          logger,
	}
}

type emptyTxIDCache struct{}

func (e *emptyTxIDCache) Contains(id []byte, ts int64) bool {
	return false
}

func (e emptyTxIDCache) Add(id []byte, ts int64) {
}

func (e emptyTxIDCache) RemoveOldTXsByTS(ts int64) {
}

func (e emptyTxIDCache) Len() int {
	return 0
}

var emptyTic = &emptyTxIDCache{}

func newEmptyTxIDCache() TXIDCache {
	return emptyTic
}
