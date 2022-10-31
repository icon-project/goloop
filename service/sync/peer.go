package sync

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type peer struct {
	lock    sync.Mutex
	id      module.PeerID
	reqID   uint32
	expired int
	timer   *time.Timer
	cb      Callback
	log     log.Logger
}

func (p *peer) onReceive(pi module.ProtocolInfo, data interface{}) bool {
	p.log.Tracef("peer.onReceive pi(%s), p(%s)\n", pi, p)
	var status errCode
	var t syncType
	if p.cb == nil {
		p.log.Warnf("Received early than setting callback")
		return false
	}
	switch pi {
	case protoResult:
		r := data.(*result)
		status = r.Status
		p.cb.onResult(status, p)
	case protoNodeData:
		var state [][]byte
		rd := data.(*nodeData)
		status = rd.Status
		t = rd.Type
		state = rd.Data
		p.cb.onNodeData(p, status, t, state)
	default:
		p.log.Warnf("Received wrong type (%s)\n", pi)
		return false
	}
	return true
}

func (p *peer) IsValidRequest(reqID uint32) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.reqID == reqID
}

func (p *peer) String() string {
	return fmt.Sprintf("peer id(%s), reqID(%d)", p.id, p.reqID)
}

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
	if e, ok := pp.peers[id]; ok == true {
		pp.pList.Remove(e)
		delete(pp.peers, id)
	}

	var ne *list.Element
	pushed := false
	for e := pp.pList.Front(); e != nil; e = e.Next() {
		lp := e.Value.(*peer)
		if p.expired < lp.expired {
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
	e := pp.peers[PeerIDToKey(id)]
	if e == nil {
		return nil
	}
	pp.pList.Remove(e)
	return e.Value.(*peer)
}

func (pp *peerPool) getPeer(id module.PeerID) *peer {
	e := pp.peers[PeerIDToKey(id)]
	if e == nil {
		return nil
	}
	return e.Value.(*peer)
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
