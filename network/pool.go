package network

import (
	"sync"
	"time"
)

type PacketPool struct {
	buckets     []map[uint64]*Packet
	len         []int
	cur         int
	numOfBucket int
	lenOfBucket int
	mtx         sync.RWMutex
}

func NewPacketPool(numOfBucket uint8, lenOfBucket uint16) *PacketPool {
	p := &PacketPool{
		buckets:     make([]map[uint64]*Packet, numOfBucket),
		len:         make([]int, numOfBucket),
		cur:         0,
		numOfBucket: int(numOfBucket),
		lenOfBucket: int(lenOfBucket),
	}
	p.buckets[0] = make(map[uint64]*Packet)
	return p
}

func (p *PacketPool) _contains(pkt *Packet) bool {
	cur := p.cur
	for i := 0; i < p.numOfBucket; i++ {
		m := p.buckets[cur]
		if m == nil {
			return false
		}
		_, ok := m[pkt.hashOfPacket]
		if ok {
			return true
		}
		if cur < 1 {
			cur = p.numOfBucket
		}
		cur--
	}
	return false
}

func (p *PacketPool) _put(pkt *Packet) bool {
	if p._contains(pkt) {
		return false
	}
	m := p.buckets[p.cur]
	m[pkt.hashOfPacket] = pkt
	p.len[p.cur]++
	if p.len[p.cur] >= p.lenOfBucket {
		p.cur++
		if p.cur >= p.numOfBucket {
			p.cur = 0
		}
		p.buckets[p.cur] = make(map[uint64]*Packet)
		p.len[p.cur] = 0
	}
	return true
}

func (p *PacketPool) Put(pkt *Packet) bool {
	defer p.mtx.Unlock()
	p.mtx.Lock()
	return p._put(pkt)
}

func (p *PacketPool) PutWith(pkt *Packet, f func(*Packet)) bool {
	defer p.mtx.Unlock()
	p.mtx.Lock()
	r := p._put(pkt)
	if r && f != nil {
		f(pkt)
	}
	return r
}

func (p *PacketPool) Clear() {
	defer p.mtx.Unlock()
	p.mtx.Lock()

	for i := 0; i < p.numOfBucket; i++ {
		p.buckets[i] = nil
	}
	p.cur = 0
	p.buckets[0] = make(map[uint64]*Packet)
}

func (p *PacketPool) Contains(pkt *Packet) bool {
	defer p.mtx.RUnlock()
	p.mtx.RLock()

	return p._contains(pkt)
}

type TimestampPool struct {
	timestamp   []int64
	buckets     []map[interface{}]interface{}
	cur         int
	numOfBucket int
	lastRemove  int64
	mtx         sync.RWMutex
}

func NewTimestampPool(numOfBucket uint8) *TimestampPool {
	p := &TimestampPool{
		timestamp:   make([]int64, numOfBucket),
		buckets:     make([]map[interface{}]interface{}, numOfBucket),
		cur:         0,
		numOfBucket: int(numOfBucket),
	}
	return p
}

func (p *TimestampPool) _contains(k interface{}) bool {
	cur := p.cur
	for i := 0; i < p.numOfBucket; i++ {
		t := p.timestamp[cur]
		if t < 1 {
			return false
		}
		m := p.buckets[cur]
		_, ok := m[k]
		if ok {
			return true
		}
		if cur < 1 {
			cur = p.numOfBucket
		}
		cur--
	}
	return false
}

func (p *TimestampPool) Put(k interface{}) {
	defer p.mtx.Unlock()
	p.mtx.Lock()

	now := time.Now()
	n := now.Unix()
	t := p.timestamp[p.cur]
	if t != n {
		p.cur++
		if p.cur >= p.numOfBucket {
			p.cur = 0
		}
		p.buckets[p.cur] = make(map[interface{}]interface{})
		p.timestamp[p.cur] = n
	}
	m := p.buckets[p.cur]
	m[k] = now
}

func (p *TimestampPool) Clear() {
	defer p.mtx.Unlock()
	p.mtx.Lock()

	for i := 0; i < p.numOfBucket; i++ {
		p.buckets[i] = nil
		p.timestamp[i] = 0
	}
	p.cur = 0
}

func (p *TimestampPool) RemoveBefore(secondDuration int) {
	defer p.mtx.Unlock()
	p.mtx.Lock()

	expire := time.Now().Unix() - int64(secondDuration)
	if p.lastRemove >= expire {
		return
	}
	cur := p.cur
	for i := 0; i < p.numOfBucket; i++ {
		t := p.timestamp[cur]
		if t <= expire {
			p.buckets[cur] = nil
			p.timestamp[cur] = 0
		}
		if cur < 1 {
			cur = p.numOfBucket
		}
		cur--
	}
	p.lastRemove = expire
}

func (p *TimestampPool) Contains(k interface{}) bool {
	defer p.mtx.RUnlock()
	p.mtx.RLock()

	return p._contains(k)
}
