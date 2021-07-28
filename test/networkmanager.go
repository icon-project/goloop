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
	t         *testing.T
	peers     []Peer
	handlers  []*nmHandler
	joinedMPI []module.ProtocolInfo
	roles     map[string]module.Role
	id        module.PeerID
	rCh       chan packetEntry
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
		t:   t,
		roles: make(map[string]module.Role),
		id:  network.NewPeerIDFromAddress(a),
		rCh: make(chan packetEntry, chLen),
	}
}

func (n *NetworkManager) attach(p Peer) {
	if indexOf(n.peers, p.ID()) < 0 {
		n.peers = append(n.peers, p)
		mpis := p.joinedProto()
		for _, mpi := range mpis {
			n.notifyJoin(p, mpi)
		}
	}
}

func (n *NetworkManager) detach(p Peer) {
	if i := indexOf(n.peers, p.ID()); i >= 0 {
		mpis := p.joinedProto()
		for _, mpi := range mpis {
			n.notifyLeave(p, mpi)
		}
		last := len(n.peers) - 1
		n.peers[i] = n.peers[last]
		n.peers[last] = nil
		n.peers = n.peers[:last]
	}
}

func (n *NetworkManager) notifyPacket(pk *Packet, cb func(rebroadcast bool, err error)) {
	Go(func() {
		for _, h := range n.handlers {
			if pk.MPI == h.mpi {
				rb, err := h.reactor.OnReceive(pk.PI, pk.Data, pk.Src)
				if cb != nil {
					cb(rb, err)
				}
				return
			}
		}
	})
}

func (n *NetworkManager) joinedProto() []module.ProtocolInfo {
	return n.joinedMPI
}

func (n *NetworkManager) notifyJoin(p Peer, mpi module.ProtocolInfo) {
	for _, h := range n.handlers {
		if h.mpi == mpi {
			h.reactor.OnJoin(p.ID())
		}
	}
}

func (n *NetworkManager) notifyLeave(p Peer, mpi module.ProtocolInfo) {
	for _, h := range n.handlers {
		if h.mpi == mpi {
			h.reactor.OnLeave(p.ID())
		}
	}
}

func (n *NetworkManager) GetPeers() []module.PeerID {
	peerIDs := make([]module.PeerID, len(n.peers))
	for i, p := range n.peers {
		peerIDs[i] = p.ID()
	}
	return peerIDs
}

func (n *NetworkManager) RegisterReactor(name string, mpi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	h := &nmHandler{
		n,
		mpi,
		name,
		reactor,
		piList,
		priority,
	}
	n.handlers = append(n.handlers, h)
	n.joinedMPI = append(n.joinedMPI, mpi)
	for _, p := range n.peers {
		for _, pmpi := range p.joinedProto() {
			if mpi == pmpi {
				id := p.ID()
				Go(func() {
					h.reactor.OnJoin(id)
				})
			}
		}
	}
	return h, nil
}

func (n *NetworkManager) RegisterReactorForStreams(name string, pi module.ProtocolInfo, reactor module.Reactor, piList []module.ProtocolInfo, priority uint8) (module.ProtocolHandler, error) {
	return n.RegisterReactor(name, pi, reactor, piList, priority)
}

func (n *NetworkManager) UnregisterReactor(reactor module.Reactor) error {
	for i, h := range n.handlers {
		if h.reactor == reactor {
			last := len(n.handlers) - 1
			n.handlers[i] = n.handlers[last]
			n.handlers[last] = nil
			n.handlers = n.handlers[:last]
			for i, mpi := range n.joinedMPI {
				if mpi == h.mpi {
					last := len(n.joinedMPI) - 1
					n.joinedMPI[i] = n.joinedMPI[last]
					n.joinedMPI[last] = 0
					n.joinedMPI = n.joinedMPI[:last]
				}
			}
			for _, p := range n.peers {
				for _, pmpi := range p.joinedProto() {
					if h.mpi == pmpi {
						p.notifyLeave(n, h.mpi)
					}
				}
			}
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
	h := p.Join(mpi)
	return p, h
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
