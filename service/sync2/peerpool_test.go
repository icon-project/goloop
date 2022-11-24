package sync2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeerPoolPush(t *testing.T) {
	pool := newPeerPool()

	// given new peer
	p1 := newPeer(createAPeerID(), nil, nil)

	// when push peer to peerpool
	pool.push(p1)

	// then pool size is 1
	expected := 1
	actual := pool.size()
	assert.EqualValuesf(t, expected, actual, "pool size expected=%d, actual=%d", expected, actual)

	// when push duplicate peer to peerpool
	pool.push(p1)

	// then
	expected = 1
	actual = pool.size()
	assert.EqualValuesf(t, expected, actual, "pool size expected=%d, actual=%d", expected, actual)

	// given peer expired +200ms
	p1.expired += 200 * time.Millisecond

	// when expired vaule of new peer less than peer in pool
	p2 := newPeer(createAPeerID(), nil, nil)
	pool.push(p2)

	// then new peer inserted before of peer in pool
	expected1 := p2.id
	actual1 := pool.pop().id
	assert.Equalf(t, expected1, actual1, "poped peer.id expected=%d, actual=%d", expected, actual)
}

func TestPeerPoolPop(t *testing.T) {
	pool := newPeerPool()

	// given peer in pool
	p1 := newPeer(createAPeerID(), nil, nil)
	pool.push(p1)

	// when pop
	popedPeer := pool.pop()

	// then
	expected := p1.id
	actual := popedPeer.id
	assert.Equalf(t, expected, actual, "pop peer.id expected=%v, acutal=%v", expected, actual)

	// assert empty peerpool
	expectedSize := 0
	actualSize := pool.size()
	assert.Equalf(t, expectedSize, actualSize, "pool size expected=%d, actual=%d", expectedSize, actualSize)

	// when pop from empty pool
	got := pool.pop()

	// then got nil
	assert.Emptyf(t, got, "got peer expected=nil, actual=%v", got)
}

func TestPeerPoolRemove(t *testing.T) {
	pool := newPeerPool()

	// given peer in pool
	p1 := newPeer(createAPeerID(), nil, nil)
	pool.push(p1)

	// when remove
	removedPeer := pool.remove(p1.id)

	// then peer == removedPeer
	expected := p1
	actual := removedPeer
	assert.Equalf(t, expected, actual, "removed peer expected=%v, actual=%v", expected, actual)

	// assert empty peerpool
	expectedSize := 0
	actualSize := pool.size()
	assert.Equalf(t, expectedSize, actualSize, "pool size expected=%d, actual=%d", expectedSize, actualSize)

	// when pop from empty pool
	removed := pool.remove(p1.id)

	// then got nil
	assert.Emptyf(t, removed, "removed peer expected=nil, actual=%v", removed)
}

func TestPeerPoolGetPeer(t *testing.T) {
	pool := newPeerPool()

	// given peer in pool
	p1 := newPeer(createAPeerID(), nil, nil)
	pool.push(p1)

	// when get peer
	gotPeer := pool.getPeer(p1.id)

	// then peer == gotPeer
	expected := p1
	actual := gotPeer
	assert.Equalf(t, expected, actual, "get peer expected=%v, actual=%v", expected, actual)

	// when get peer unknown peer id
	gotPeer = pool.getPeer(createAPeerID())

	// then got nil
	assert.Emptyf(t, gotPeer, "get peer with unknown peer id expected=nil, actual=%v", gotPeer)
}

func TestPeerPoolPeerList(t *testing.T) {
	pool := newPeerPool()

	// when pool is empty
	peers := pool.peerList()

	// then len(peers) is 0
	expectedSize := 0
	actualSize := len(peers)
	assert.EqualValuesf(t, expectedSize, actualSize, "peerList size expected=%d, actual=%d", expectedSize, actualSize)

	// given peer in pool
	p1 := newPeer(createAPeerID(), nil, nil)
	pool.push(p1)

	// when get peer list
	peers = pool.peerList()

	// then poolsize == len(peers)
	expectedSize = pool.size()
	actualSize = len(peers)
	assert.EqualValuesf(t, expectedSize, actualSize, "peerList size expected=%d, actual=%d", expectedSize, actualSize)
}

func TestPeerPoolClear(t *testing.T) {
	pool := newPeerPool()

	// given 10 peers in pool
	const PoolSize = 10
	for range [PoolSize]int{} {
		p1 := newPeer(createAPeerID(), nil, nil)
		pool.push(p1)
	}

	// assert pool size
	expectedSize := PoolSize
	actualSize := pool.size()
	assert.EqualValuesf(t, expectedSize, actualSize, "peerList size expected=%d, actual=%d", expectedSize, actualSize)

	// when clear
	pool.clear()

	// then pool size is 0
	expectedSize = 0
	actualSize = pool.size()
	assert.EqualValuesf(t, expectedSize, actualSize, "peerList size expected=%d, actual=%d", expectedSize, actualSize)
}

func TestPeerPoolHas(t *testing.T) {
	type tcData struct {
		given    func(p *peer, pool *peerPool)
		expected bool
	}

	tests := map[string]tcData{
		"peerExists": {
			given: func(p *peer, pool *peerPool) {
				pool.push(p)
			},
			expected: true,
		},
		"peerNotExists": {
			given: func(p *peer, pool *peerPool) {
				// nothing to do
			},
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p := newPeer(createAPeerID(), nil, nil)
			pool := newPeerPool()

			// given peer in pool
			tc.given(p, pool)

			// when check peer id
			hasPeer := pool.has(p.id)

			// then
			assert.EqualValuesf(t, tc.expected, hasPeer, "pool has peer expected=%v, actual=%v", tc.expected, hasPeer)
		})
	}
}
