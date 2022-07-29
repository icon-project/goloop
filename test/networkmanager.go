/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package test

import (
	"testing"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

type NetworkManager struct {
	module.NetworkManager
	t        *testing.T
	peers    []Peer
	handlers []*nmHandler
	roles    map[string]module.Role
	id       module.PeerID
	rCh      chan packetEntry
}

func indexOf(pl []Peer, id module.PeerID) int {
	for i := range pl {
		if pl[i].ID().Equal(id) {
			return i
		}
	}
	return -1
}

func NewNetworkManager(t *testing.T, a module.Address) *NetworkManager {
	const chLen = 1024
	return &NetworkManager{
		t:     t,
		roles: make(map[string]module.Role),
		id:    network.NewPeerIDFromAddress(a),
		rCh:   make(chan packetEntry, chLen),
	}
}

func (n *NetworkManager) attach(p Peer) {
	if indexOf(n.peers, p.ID()) < 0 {
		n.peers = append(n.peers, p)
		n.notifyJoin(p)
	}
}

func (n *NetworkManager) detach(p Peer) {
	if i := indexOf(n.peers, p.ID()); i >= 0 {
		n.notifyLeave(p)
		last := len(n.peers) - 1
		n.peers[i] = n.peers[last]
		n.peers[last] = nil
		n.peers = n.peers[:last]
	}
}

func (n *NetworkManager) notifyPacket(pk *Packet, cb func(rebroadcast bool, err error)) {
	for _, h := range n.handlers {
		if pk.MPI == h.mpi {
			reactor := h.reactor
			Go(func() {
				rb, err := reactor.OnReceive(pk.PI, pk.Data, pk.Src)
				if cb != nil {
					cb(rb, err)
				}
			})
			return
		}
	}
}

func (n *NetworkManager) notifyJoin(p Peer) {
	for _, h := range n.handlers {
		reactor := h.reactor
		Go(func() {
			reactor.OnJoin(p.ID())
		})
	}
}

func (n *NetworkManager) notifyLeave(p Peer) {
	for _, h := range n.handlers {
		reactor := h.reactor
		Go(func() {
			reactor.OnLeave(p.ID())
		})
	}
}

func (n *NetworkManager) GetPeers() []module.PeerID {
	peerIDs := make([]module.PeerID, len(n.peers))
	for i, p := range n.peers {
		peerIDs[i] = p.ID()
	}
	return peerIDs
}

func (n *NetworkManager) RegisterReactor(name string, mpi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	h := &nmHandler{
		n,
		mpi,
		name,
		reactor,
		piList,
		priority,
	}
	n.handlers = append(n.handlers, h)
	return h, nil
}

func (n *NetworkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8, policy module.NotRegisteredProtocolPolicy) (module.ProtocolHandler, error) {
	return n.RegisterReactor(name, pi, reactor, piList, priority, policy)
}

func (n *NetworkManager) UnregisterReactor(reactor module.Reactor) error {
	for i, h := range n.handlers {
		if h.reactor == reactor {
			last := len(n.handlers) - 1
			n.handlers[i] = n.handlers[last]
			n.handlers[last] = nil
			n.handlers = n.handlers[:last]
			return nil
		}
	}
	return nil
}

func (n *NetworkManager) SetRole(version int64, role module.Role, peers ...module.PeerID) {
	for k, v := range n.roles {
		if v == role {
			delete(n.roles, k)
		}
	}
	for _, id := range peers {
		n.roles[string(id.Bytes())] = role
	}
}

func (n *NetworkManager) NewPeerFor(mpi module.ProtocolInfo) (*SimplePeer, *SimplePeerHandler) {
	p := NewPeer(n.t).Connect(n)
	h := p.RegisterProto(mpi)
	return p, h
}

func (n *NetworkManager) NewPeerForWithAddress(mpi module.ProtocolInfo, w module.Wallet) (*SimplePeer, *SimplePeerHandler) {
	p := NewPeerWithAddress(n.t, w).Connect(n)
	h := p.RegisterProto(mpi)
	return p, h
}

func (n *NetworkManager) Connect(n2 *NetworkManager) {
	PeerConnect(n, n2)
}

func (n *NetworkManager) ID() module.PeerID {
	return n.id
}

type nmHandler struct {
	n        *NetworkManager
	mpi      module.ProtocolInfo
	name     string
	reactor  module.Reactor
	piList   []module.ProtocolInfo
	priority uint8
}

func (h *nmHandler) Broadcast(pi module.ProtocolInfo, b []byte, bt module.BroadcastType) error {
	for _, p := range h.n.peers {
		pk := &Packet{
			SendTypeBroadcast,
			h.n.id,
			bt,
			h.mpi,
			pi,
			b,
		}
		p.notifyPacket(pk, nil)
	}
	return nil
}

func (h *nmHandler) Multicast(pi module.ProtocolInfo, b []byte, role module.Role) error {
	for _, p := range h.n.peers {
		if h.n.roles[string(p.ID().Bytes())] == role {
			pk := &Packet{
				SendTypeMulticast,
				h.n.id,
				role,
				h.mpi,
				pi,
				b,
			}
			p.notifyPacket(pk, nil)
		}
	}
	return nil
}

func (h *nmHandler) Unicast(pi module.ProtocolInfo, b []byte, id module.PeerID) error {
	if idx := indexOf(h.n.peers, id); idx >= 0 {
		pk := &Packet{
			SendTypeUnicast,
			h.n.id,
			id,
			h.mpi,
			pi,
			b,
		}
		h.n.peers[idx].notifyPacket(pk, nil)
		return nil
	}
	return errors.New("no peer")
}

func (h *nmHandler) GetPeers() []module.PeerID {
	return h.n.GetPeers()
}
