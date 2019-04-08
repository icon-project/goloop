package test

import (
	"runtime"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/pkg/errors"
)

type tReactorItem struct {
	name     string
	reactor  module.Reactor
	piList   []module.ProtocolInfo
	priority uint8
}

type tPacket struct {
	pi module.ProtocolInfo
	b  []byte
	id module.PeerID
}

type tNetworkManagerStatic struct {
	common.Mutex
	procList []*NetworkManager
}

var nms = tNetworkManagerStatic{}

type NetworkManager struct {
	*tNetworkManagerStatic
	ID       module.PeerID
	wakeUpCh chan struct{}

	reactorItems []*tReactorItem
	peers        []*NetworkManager
	drop         bool
	recvBuf      []*tPacket
}

type tProtocolHandler struct {
	nm *NetworkManager
	ri *tReactorItem
}

func NewNetworkManagerForPeerID(id module.PeerID) *NetworkManager {
	nm := &NetworkManager{
		tNetworkManagerStatic: &nms,
		ID:                    id,
		wakeUpCh:              make(chan struct{}, 1),
	}
	go nm.process()
	runtime.SetFinalizer(nm, (*NetworkManager).dispose)
	return nm
}

func (nm *NetworkManager) dispose() {
	close(nm.wakeUpCh)
}

func NewNetworkManager() *NetworkManager {
	return NewNetworkManagerForPeerID(createAPeerID())
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

func (nm *NetworkManager) RegisterReactor(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
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

func (nm *NetworkManager) RegisterReactorForStreams(name string, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	return nm.RegisterReactor(name, reactor, piList, priority)
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

func (nm *NetworkManager) onReceiveUnicast(pi module.ProtocolInfo, b []byte, from module.PeerID) {
	nm.Lock()
	nm.recvBuf = append(nm.recvBuf, &tPacket{pi, b, from})
	nm.Unlock()
	select {
	case nm.wakeUpCh <- struct{}{}:
	default:
	}
}

func (nm *NetworkManager) process() {
	for {
		select {
		case _, more := <-nm.wakeUpCh:
			if !more {
				return
			}
		}
		nm.Lock()
		recvBuf := nm.recvBuf
		nm.recvBuf = nil
		reactorItems := make([]*tReactorItem, len(nm.reactorItems))
		copy(reactorItems, nm.reactorItems)
		nm.Unlock()
		for _, p := range recvBuf {
			for _, r := range reactorItems {
				r.reactor.OnReceive(p.pi, p.b, p.id)
			}
		}
	}
}

func (ph *tProtocolHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	panic("not implemented")
}

func (ph *tProtocolHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	ph.nm.Lock()
	defer ph.nm.Unlock()

	if ph.nm.drop {
		return nil
	}
	for _, p := range ph.nm.peers {
		if p.ID.Equal(id) {
			peer := p
			id := ph.nm.ID
			ph.nm.CallAfterUnlock(func() {
				peer.onReceiveUnicast(pi, b, id)
			})
			return nil
		}
	}
	return errors.Errorf("Unknown peer")
}

func createAPeerID() module.PeerID {
	return network.NewPeerIDFromAddress(wallet.New().Address())
}
