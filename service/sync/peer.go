package sync

import (
	"container/list"
	"fmt"
	"time"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type peer struct {
	id    module.PeerID
	reqID uint32
	timer *time.Timer
	cb    Callback
}

func (p *peer) onReceive(r int, pi module.ProtocolInfo, data interface{}) {
	log.Debugf("peer.onReceive result(%d), pi(%s), p(%s)\n", r, pi, p)
	var status errCode
	var t syncType
	switch pi {
	case protoResult:
		if r == receiveMsg {
			r := data.(*result)
			status = r.Status
		}
		p.cb.onResult(status, p)
	case protoNodeData:
		var state [][]byte
		if r == receiveMsg {
			rd := data.(*nodeData)
			status = rd.Status
			t = rd.Type
			state = rd.Data
		}
		p.cb.onNodeData(p, status, t, state)
	}
}

func (p *peer) String() string {
	return fmt.Sprintf("peer id(%s), reqID(%d)\n", p.id, p.reqID)
}

type peerPool struct {
	ch    chan module.PeerID
	peers map[module.PeerID]*list.Element
	pList *list.List //peer
}

func newPeerPool() *peerPool {
	return &peerPool{
		ch:    make(chan module.PeerID),
		peers: make(map[module.PeerID]*list.Element),
		pList: list.New(),
	}
}

func (pp *peerPool) push(id module.PeerID, p *peer) *peer {
	if e, ok := pp.peers[id]; ok == true {
		pp.pList.Remove(e)
		delete(pp.peers, id)
	}

	np := p
	if p == nil {
		np = &peer{
			id:    id,
			reqID: 0,
		}
	}
	e := pp.pList.PushBack(np)
	pp.peers[id] = e
	log.Debugf("peerPool push(%s), len(%d)\n", np, pp.pList.Len())
	return np
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
	log.Debugf("peerList len(%d)\n", pp.pList.Len())
	return pList
}
