package sync

import (
	"container/list"
	"fmt"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type peer struct {
	id      module.PeerID
	reqID   uint32
	expired time.Duration
	timer   *time.Timer
	cb      Callback
	log     log.Logger
}

func (p *peer) onReceive(pi module.ProtocolInfo, data interface{}) bool {
	p.log.Tracef("peer.onReceive pi(%s), p(%s)\n", pi, p)
	var status errCode
	var t syncType
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
		p.log.Info("Received wrong type (%s)\n", pi)
		return false
	}
	return true
}

func (p *peer) String() string {
	return fmt.Sprintf("peer id(%s), reqID(%d)", p.id, p.reqID)
}

type peerPool struct {
	ch    chan module.PeerID
	peers map[module.PeerID]*list.Element
	pList *list.List //peer
	log   log.Logger
}

func newPeerPool(log log.Logger) *peerPool {
	return &peerPool{
		ch:    make(chan module.PeerID),
		peers: make(map[module.PeerID]*list.Element),
		pList: list.New(),
		log:   log,
	}
}

func (pp *peerPool) push(p *peer) {
	id := p.id
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
	p.log.Tracef("peerPool push(%s), len(%d)\n", p, pp.pList.Len())
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
	delete(pp.peers, peer.id)
	return peer
}

func (pp *peerPool) remove(id module.PeerID) {
	e := pp.peers[id]
	if e == nil {
		return
	}
	pp.pList.Remove(e)
}

func (pp *peerPool) getPeer(id module.PeerID) *peer {
	e := pp.peers[id]
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
	pp.log.Tracef("peerList len(%d)\n", pp.pList.Len())
	return pList
}
