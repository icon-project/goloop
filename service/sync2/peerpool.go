package sync2

import (
	"container/list"

	"github.com/icon-project/goloop/module"
)

type peerPool struct {
	peers map[string]*list.Element
	pList *list.List // peer
}

func newPeerPool() *peerPool {
	return &peerPool{
		peers: make(map[string]*list.Element),
		pList: list.New(),
	}
}

func PeerIDToKey(p module.PeerID) string {
	return string(p.Bytes())
}

func (pp *peerPool) push(p *peer) {
	id := PeerIDToKey(p.id)
	if e, ok := pp.peers[id]; ok {
		pp.pList.Remove(e)
		delete(pp.peers, id)
	}

	var ne *list.Element
	pushed := false
	for e := pp.pList.Front(); e != nil; e = e.Next() {
		lp := e.Value.(*peer)
		if p.getExpired() < lp.getExpired() {
			ne = pp.pList.InsertBefore(p, e)
			pushed = true
			break
		}
	}
	if !pushed {
		ne = pp.pList.PushBack(p)
	}

	pp.peers[id] = ne
}

func (pp *peerPool) size() int {
	return pp.pList.Len()
}

func (pp *peerPool) pop() *peer {
	if pp.pList.Len() == 0 {
		return nil
	}
	e := pp.pList.Front()
	peer := e.Value.(*peer)
	pp.pList.Remove(e)
	delete(pp.peers, PeerIDToKey(peer.id))
	return peer
}

func (pp *peerPool) remove(id module.PeerID) *peer {
	key := PeerIDToKey(id)
	if e, ok := pp.peers[key]; ok {
		pp.pList.Remove(e)
		delete(pp.peers, key)
		return e.Value.(*peer)
	}
	return nil
}

func (pp *peerPool) getPeer(id module.PeerID) *peer {
	if e, ok := pp.peers[PeerIDToKey(id)]; ok {
		return e.Value.(*peer)
	}
	return nil
}

func (pp *peerPool) peerList() []*peer {
	pList := make([]*peer, pp.pList.Len())
	i := 0
	for e := pp.pList.Front(); e != nil; e = e.Next() {
		pList[i] = e.Value.(*peer)
		i++
	}
	return pList
}

func (pp *peerPool) clear() {
	pp.pList.Init()
	pp.peers = make(map[string]*list.Element)
}

func (pp *peerPool) has(id module.PeerID) bool {
	_, ok := pp.peers[PeerIDToKey(id)]
	return ok
}
