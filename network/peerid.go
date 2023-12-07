package network

import (
	"bytes"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	peerIDSize      = 20 // common.AddressIDBytes
	peerIDCacheSize = 100
)

type peerID struct {
	module.Address
	next  *peerID
	pPrev **peerID
}

func (pi *peerID) Bytes() []byte {
	return pi.Address.ID()
}

func (pi *peerID) Equal(a module.PeerID) bool {
	return bytes.Equal(pi.Bytes(), a.Bytes())
}

func (pi *peerID) String() string {
	return pi.Address.String()
}

type peerIDCache struct {
	cache map[string]*peerID
	front *peerID
	pLast **peerID
	len   int
	size  int
	lock  sync.Mutex
}

func (c *peerIDCache) remove(p *peerID) {
	if p.pPrev == nil {
		return
	}
	*p.pPrev = p.next
	if p.next == nil {
		c.pLast = p.pPrev
	} else {
		p.next.pPrev = p.pPrev
	}
	p.pPrev = nil
	p.next = nil
	c.len -= 1
}

func (c *peerIDCache) add(p *peerID) {
	*c.pLast = p
	p.pPrev = c.pLast
	c.pLast = &p.next
	c.len += 1
}

func (c *peerIDCache) moveToBack(p *peerID) {
	if p.next == nil {
		return
	}
	c.remove(p)
	c.add(p)
}

func (c *peerIDCache) Get(addr module.Address) module.PeerID {
	c.lock.Lock()
	defer c.lock.Unlock()
	ks := string(addr.Bytes())
	if p, ok := c.cache[ks]; ok {
		c.moveToBack(p)
		return p
	}
	if c.len == c.size {
		delete(c.cache, string(c.front.Bytes()))
		c.remove(c.front)
	}
	p := &peerID{
		Address: addr,
	}
	c.cache[ks] = p
	c.add(p)
	return p
}

func newPeerIDCache(size int) *peerIDCache {
	c := &peerIDCache{
		cache: make(map[string]*peerID),
		len:   0,
		size:  size,
	}
	c.pLast = &c.front
	return c
}

var cache = newPeerIDCache(peerIDCacheSize)

func NewPeerID(b []byte) module.PeerID {
	return cache.Get(common.NewAccountAddress(b))
}

func NewPeerIDFromAddress(a module.Address) module.PeerID {
	return cache.Get(a)
}

func NewPeerIDFromPublicKey(k *crypto.PublicKey) module.PeerID {
	return cache.Get(common.NewAccountAddressFromPublicKey(k))
}

func NewPeerIDFromString(s string) (module.PeerID, error) {
	a, err := common.NewAddressFromString(s)
	if err != nil {
		return nil, err
	}
	if a.IsContract() {
		return nil, errors.Errorf("invalid PeerID:%s", s)
	}
	return cache.Get(a), nil
}
