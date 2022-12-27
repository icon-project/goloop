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

type channelPeerHandler struct {
	ph  PeerHandler
	mtr *metric.NetworkMetric
}

type PeerDispatcher struct {
	*peerHandler
	peerHandlers      *list.List
	peerHandlersMtx   sync.RWMutex
	peerHandlerMap    map[string]*channelPeerHandler
	peerHandlerMapMtx sync.RWMutex

	mtr *metric.NetworkMetric
}

func newPeerDispatcher(id module.PeerID, l log.Logger, peerHandlers ...PeerHandler) *PeerDispatcher {
	pd := &PeerDispatcher{
		peerHandlers:   list.New(),
		peerHandlerMap: make(map[string]*channelPeerHandler),
		peerHandler:    newPeerHandler(id, l),
		mtr:            metric.NewNetworkMetric(metric.DefaultMetricContext()),
	}

	//listener or dialer => pd.dispatchPeer => front.onPeer => back.onPeer => pd.onPeer => p2p.onPeer
	for _, ph := range peerHandlers {
		pd.registerPeerHandler(ph, true)
	}
	return pd
}

func (pd *PeerDispatcher) registerByChannel(channel string, ph PeerHandler, mtr *metric.NetworkMetric) bool {
	pd.peerHandlerMapMtx.Lock()
	defer pd.peerHandlerMapMtx.Unlock()

	if _, ok := pd.peerHandlerMap[channel]; ok {
		return false
	}
	pd.peerHandlerMap[channel] = &channelPeerHandler{
		ph:  ph,
		mtr: mtr,
	}
	return true
}

func (pd *PeerDispatcher) unregisterByChannel(channel string) bool {
	pd.peerHandlerMapMtx.Lock()
	defer pd.peerHandlerMapMtx.Unlock()

	_, ok := pd.peerHandlerMap[channel]
	if ok {
		delete(pd.peerHandlerMap, channel)
	}
	return ok
}

func (pd *PeerDispatcher) getByChannel(channel string) (*channelPeerHandler, bool) {
	pd.peerHandlerMapMtx.RLock()
	defer pd.peerHandlerMapMtx.RUnlock()

	v, ok := pd.peerHandlerMap[channel]
	return v, ok
}

func (pd *PeerDispatcher) registerPeerHandler(ph PeerHandler, pushBack bool) {
	pd.peerHandlersMtx.Lock()
	defer pd.peerHandlersMtx.Unlock()

	pd.logger.Traceln("registerPeerHandler", ph, pushBack)
	if pushBack {
		if back := pd.peerHandlers.Back(); back != nil {
			back.Value.(PeerHandler).setNext(ph)
		}
		pd.peerHandlers.PushBack(ph)
		ph.setNext(pd)
	} else {
		if front := pd.peerHandlers.Front(); front != nil {
			ph.setNext(front.Value.(PeerHandler))
		} else {
			ph.setNext(pd)
		}
		pd.peerHandlers.PushFront(ph)
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
	pd.peerHandlersMtx.RLock()
	defer pd.peerHandlersMtx.RUnlock()

	front := pd.peerHandlers.Front()
	ph := front.Value.(PeerHandler)
	p.setMetric(pd.mtr)
	p.setPacketCbFunc(ph.onPacket)
	p.setCloseCbFunc(ph.onClose)
	ph.onPeer(p)
}

//callback from PeerHandler.nextOnPeer
func (pd *PeerDispatcher) onPeer(p *Peer) {
	pd.logger.Traceln("onPeer", p)
	if v, ok := pd.getByChannel(p.Channel()); ok {
		p.setMetric(v.mtr)
		p.setPacketCbFunc(v.ph.onPacket)
		p.setCloseCbFunc(v.ph.onClose)
		v.ph.onPeer(p)
	} else {
		err := fmt.Errorf("not exists PeerToPeer[%s]", p.Channel())
		p.CloseByError(err)
	}
}

//callback from Peer.receiveRoutine
func (pd *PeerDispatcher) onPacket(pkt *Packet, p *Peer) {
	pd.logger.Traceln("onPacket", pkt)
}

func (pd *PeerDispatcher) onClose(p *Peer) {
	pd.peerHandler.onClose(p)
}
