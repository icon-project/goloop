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

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

type SimplePeer struct {
	t         *testing.T
	id        module.PeerID
	p         Peer
	handlers  []*SimplePeerHandler
	joinedMPI []module.ProtocolInfo
	w         module.Wallet
}

func NewPeer(t *testing.T) *SimplePeer {
	w := wallet.New()
	return &SimplePeer{
		t:  t,
		id: network.NewPeerIDFromAddress(w.Address()),
		w:  w,
	}
}

func NewPeerWithAddress(t *testing.T, w module.Wallet) *SimplePeer {
	return &SimplePeer{
		t:  t,
		id: network.NewPeerIDFromAddress(w.Address()),
		w:  w,
	}
}

func (p *SimplePeer) Wallet() module.Wallet {
	return p.w
}

func (p *SimplePeer) Address() module.Address {
	return p.w.Address()
}

func (p *SimplePeer) attach(p2 Peer) {
	p.p = p2
}

func (p *SimplePeer) detach(p2 Peer) {
	p.p = nil
}

func (p *SimplePeer) notifyPacket(pk *Packet, cb func(rebroadcast bool, err error)) {
	for _, h := range p.handlers {
		if h.mpi == pk.MPI {
			h.rCh <- packetEntry{pk, cb}
		}
	}
}

func (p *SimplePeer) joinedProto() []module.ProtocolInfo {
	return p.joinedMPI
}

func (p *SimplePeer) notifyJoin(p2 Peer, mpi module.ProtocolInfo) {
}

func (p *SimplePeer) notifyLeave(p2 Peer, mpi module.ProtocolInfo) {
}

func (p *SimplePeer) ID() module.PeerID {
	return p.id
}

func (p *SimplePeer) Connect(p2 Peer) *SimplePeer {
	p2.attach(p)
	p.attach(p2)
	return p
}

func (p *SimplePeer) Join(mpi module.ProtocolInfo) *SimplePeerHandler {
	const chanSize = 1024
	p.p.notifyJoin(p, mpi)
	h := &SimplePeerHandler{
		p:   p,
		mpi: mpi,
		rCh: make(chan packetEntry, chanSize),
	}
	p.handlers = append(p.handlers, h)
	p.joinedMPI = append(p.joinedMPI, mpi)
	return h
}

type SimplePeerHandler struct {
	p   *SimplePeer
	mpi module.ProtocolInfo
	rCh chan packetEntry
}

func (h *SimplePeerHandler) Wallet() module.Wallet {
	return h.p.w
}

func (h *SimplePeerHandler) Address() module.Address {
	return h.p.w.Address()
}

func (h *SimplePeerHandler) Peer() *SimplePeer {
	return h.p
}

func (h *SimplePeerHandler) Unicast(
	pi module.ProtocolInfo,
	m interface{},
	cb func(bool, error),
) {
	bs := codec.MustMarshalToBytes(m)
	pk := &Packet{
		SendTypeUnicast,
		h.p.id,
		h.p.p.ID(),
		h.mpi,
		pi,
		bs,
	}
	h.p.p.notifyPacket(pk, cb)
}

func (h *SimplePeerHandler) AssertReceiveUnicast(pi module.ProtocolInfo, m interface{}) {
	pe := <-h.rCh
	assert.Equal(h.p.t, SendTypeUnicast, pe.pk.SendType)
	bs := codec.MustMarshalToBytes(m)
	assert.Equal(h.p.t, bs, pe.pk.Data)
	assert.Equal(h.p.t, pi, pe.pk.PI)
}

func (h *SimplePeerHandler) Receive(
	pi module.ProtocolInfo,
	expMsg interface{},
	outMsg interface{},
) *Packet {
	pe := <-h.rCh
	assert.Equal(h.p.t, pi, pe.pk.PI)
	if expMsg != nil {
		bs := codec.MustMarshalToBytes(expMsg)
		assert.Equal(h.p.t, bs, pe.pk.Data)
	}
	_, err := codec.UnmarshalFromBytes(pe.pk.Data, outMsg)
	assert.NoError(h.p.t, err)
	return pe.pk
}

func (h *SimplePeerHandler) Multicast(
	pi module.ProtocolInfo,
	m interface{},
	role module.Role,
	cb func(bool, error),
) {
	bs := codec.MustMarshalToBytes(m)
	pk := &Packet{
		SendTypeMulticast,
		h.p.id,
		role,
		h.mpi,
		pi,
		bs,
	}
	h.p.p.notifyPacket(pk, cb)
}

func (h *SimplePeerHandler) Broadcast(
	pi module.ProtocolInfo,
	m interface{},
	btype module.BroadcastType,
	cb func(bool, error),
) {
	bs := codec.MustMarshalToBytes(m)
	pk := &Packet{
		SendTypeBroadcast,
		h.p.id,
		btype,
		h.mpi,
		pi,
		bs,
	}
	h.p.p.notifyPacket(pk, cb)
}
