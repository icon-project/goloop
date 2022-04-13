package network

import (
	"container/list"
	"fmt"
	"net"
	"sync"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/metric"
)

type PeerDispatcher struct {
	*peerHandler
	peerHandlers *list.List
	p2pMap       map[string]*PeerToPeer
	mtx          sync.RWMutex

	mtr *metric.NetworkMetric
}

func newPeerDispatcher(id module.PeerID, l log.Logger, peerHandlers ...PeerHandler) *PeerDispatcher {
	pd := &PeerDispatcher{
		peerHandlers: list.New(),
		p2pMap:       make(map[string]*PeerToPeer),
		peerHandler:  newPeerHandler(l),
		mtr:          metric.NewNetworkMetric(metric.DefaultMetricContext()),
	}
	// pd.peerHandler.codecHandle.MapType = reflect.TypeOf(map[string]interface{}(nil))
	pd.setSelfPeerID(id)

	pd.registerPeerHandler(pd, true)
	for _, ph := range peerHandlers {
		pd.registerPeerHandler(ph, true)
	}
	return pd
}

func (pd *PeerDispatcher) registerPeerToPeer(p2p *PeerToPeer) bool {
	defer pd.mtx.Unlock()
	pd.mtx.Lock()

	if _, ok := pd.p2pMap[p2p.channel]; ok {
		return false
	}
	pd.p2pMap[p2p.channel] = p2p
	return true
}

func (pd *PeerDispatcher) unregisterPeerToPeer(p2p *PeerToPeer) bool {
	defer pd.mtx.Unlock()
	pd.mtx.Lock()
	if t, ok := pd.p2pMap[p2p.channel]; !ok || t != p2p {
		return false
	}
	delete(pd.p2pMap, p2p.channel)
	return true
}

func (pd *PeerDispatcher) getPeerToPeer(channel string) *PeerToPeer {
	defer pd.mtx.RUnlock()
	pd.mtx.RLock()

	return pd.p2pMap[channel]
}

func (pd *PeerDispatcher) registerPeerHandler(ph PeerHandler, pushBack bool) {
	pd.logger.Traceln("registerPeerHandler", ph, pushBack)
	if pushBack {
		elm := pd.peerHandlers.PushBack(ph)
		if prev := elm.Prev(); prev != nil {
			ph.setNext(prev.Value.(PeerHandler))
			ph.setSelfPeerID(pd.self)
		}
	} else {
		f := pd.peerHandlers.Front()
		elm := pd.peerHandlers.InsertAfter(ph, f)
		pd.setNext(ph)
		ph.setSelfPeerID(pd.self)
		if next := elm.Next(); next != nil {
			next.Value.(PeerHandler).setNext(ph)
		}
	}
}

//callback from Listener.acceptRoutine
func (pd *PeerDispatcher) onAccept(conn net.Conn) {
	pd.logger.Traceln("onAccept", conn.LocalAddr(), "<-", conn.RemoteAddr())
	p := newPeer(conn, nil, true, "", pd.logger)
	pd.dispatchPeer(p)
}

//callback from Dialer.Connect
func (pd *PeerDispatcher) onConnect(conn net.Conn, addr string, d *Dialer) {
	pd.logger.Traceln("onConnect", conn.LocalAddr(), "->", conn.RemoteAddr())
	p := newPeer(conn, nil, false, NetAddress(addr), pd.logger)
	p.setChannel(d.channel)
	p.setNetAddress(NetAddress(addr))
	pd.dispatchPeer(p)
}

func (pd *PeerDispatcher) dispatchPeer(p *Peer) {
	elm := pd.peerHandlers.Back()
	ph := elm.Value.(PeerHandler)
	p.setMetric(pd.mtr)
	p.setPacketCbFunc(ph.onPacket)
	p.setErrorCbFunc(ph.onError)
	p.setCloseCbFunc(ph.onClose)
	ph.onPeer(p)
}

//callback from PeerHandler.nextOnPeer
func (pd *PeerDispatcher) onPeer(p *Peer) {
	pd.logger.Traceln("onPeer", p)
	if p2p := pd.getPeerToPeer(p.Channel()); p2p != nil {
		p.setMetric(p2p.mtr)
		p.setPacketCbFunc(p2p.onPacket)
		p.setErrorCbFunc(p2p.onError)
		p.setCloseCbFunc(p2p.onClose)
		p2p.onPeer(p)
	} else {
		err := fmt.Errorf("not exists PeerToPeer[%s]", p.Channel())
		p.CloseByError(err)
	}
}

func (pd *PeerDispatcher) onError(err error, p *Peer, pkt *Packet) {
	pd.peerHandler.onError(err, p, pkt)
}

//callback from Peer.receiveRoutine
func (pd *PeerDispatcher) onPacket(pkt *Packet, p *Peer) {
	//TODO dispatcher.message_dump
	pd.logger.Traceln("onPacket", pkt)
}

func (pd *PeerDispatcher) onClose(p *Peer) {
	pd.peerHandler.onClose(p)
}
