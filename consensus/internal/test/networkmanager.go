package test

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

type tReactorItem struct {
	name     string
	reactor  module.Reactor
	piList   []module.ProtocolInfo
	priority uint8
}

type tNetworkManagerStatic struct {
	common.Mutex
	jobs chan func()
}

func (p *tNetworkManagerStatic) process() {
	for {
		job := <-p.jobs
		job()
	}
}

var nms = tNetworkManagerStatic{}

type NetworkManager struct {
	module.NetworkManager
	*tNetworkManagerStatic
	ID module.PeerID

	reactorItems []*tReactorItem
	peers        []*NetworkManager
	drop         bool
}

type tProtocolHandler struct {
	nm *NetworkManager
	ri *tReactorItem
}

func newTNetworkManagerForPeerID(id module.PeerID) *NetworkManager {
	nms.Lock()
	if nms.jobs == nil {
		nms.jobs = make(chan func(), 1000)
		go nms.process()
	}
	nms.Unlock()
	nm := &NetworkManager{
		tNetworkManagerStatic: &nms,
		ID:                    id,
	}
	return nm
}

func NewNetworkManager() *NetworkManager {
	return newTNetworkManagerForPeerID(createAPeerID())
}

func (nm *NetworkManager) GetPeers() []module.PeerID {
	nm.Lock()
	defer nm.Unlock()

	res := make([]module.PeerID, len(nm.peers))
	for i := range nm.peers {
		res[i] = nm.peers[i].ID
	}
	return res
}

func (nm *NetworkManager) RegisterReactor(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	nm.Lock()
	defer nm.Unlock()

	r := &tReactorItem{
		name:     name,
		reactor:  reactor,
		piList:   piList,
		priority: priority,
	}
	nm.reactorItems = append(nm.reactorItems, r)
	return &tProtocolHandler{nm, r}, nil
}

func (nm *NetworkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	return nm.RegisterReactor(name, pi, reactor, piList, priority, policy)
}

func (nm *NetworkManager) Join(nm2 *NetworkManager) {
	nm.Lock()
	defer nm.Unlock()

	nm.peers = append(nm.peers, nm2)
	nm2.peers = append(nm2.peers, nm)
	ri := make([]*tReactorItem, len(nm.reactorItems))
	copy(ri, nm.reactorItems)
	id2 := nm2.ID
	ri2 := make([]*tReactorItem, len(nm2.reactorItems))
	copy(ri2, nm.reactorItems)
	id := nm.ID
	nm.CallAfterUnlock(func() {
		for _, r := range ri {
			r.reactor.OnJoin(id2)
		}
		for _, r := range ri2 {
			r.reactor.OnJoin(id)
		}
	})
}

func (nm *NetworkManager) receivePacket(pi module.ProtocolInfo, b []byte, from module.PeerID) {
	for _, ri := range nm.reactorItems {
		if ri.accept(pi) {
			r := ri.reactor
			nm.jobs <- func() {
				r.OnReceive(pi, b, from)
			}
		}
	}
}

func (ph *tProtocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	ph.nm.Lock()
	defer ph.nm.Unlock()

	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		p.receivePacket(pi, b, ph.nm.ID)
	}
	return nil
}

func (ph *tProtocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	ph.nm.Lock()
	defer ph.nm.Unlock()

	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		p.receivePacket(pi, b, ph.nm.ID)
	}
	return nil
}

func (ph *tProtocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	ph.nm.Lock()
	defer ph.nm.Unlock()

	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		if p.ID.Equal(id) {
			p.receivePacket(pi, b, ph.nm.ID)
			return nil
		}
	}
	return errors.Errorf("Unknown peer")
}

func (ph *tProtocolHandler) GetPeers() []module.PeerID {
	return ph.nm.GetPeers()
}

func createAPeerID() module.PeerID {
	return network.NewPeerIDFromAddress(wallet.New().Address())
}

func (ri *tReactorItem) accept(pi module.ProtocolInfo) bool {
	for _, rpi := range ri.piList {
		if pi.Uint16() == rpi.Uint16() {
			return true
		}
	}
	return false
}
