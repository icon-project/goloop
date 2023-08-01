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
	"sync"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
)

type SimplePeer struct {
	// immutable
	t T
	id module.PeerID
	w  module.Wallet

	// mutable
	mu       sync.Mutex
	p        Peer
	handlers []*SimplePeerHandler
}

func NewPeer(t T) *SimplePeer {
	w := wallet.New()
	return &SimplePeer{
		t:  t,
		id: network.NewPeerIDFromAddress(w.Address()),
		w:  w,
	}
}

func NewPeerWithAddress(t T, w module.Wallet) *SimplePeer {
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
	p.mu.Lock()
	defer p.mu.Unlock()
	p.p = p2
}

func (p *SimplePeer) detach(p2 Peer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.p = nil
}

func (p *SimplePeer) notifyPacket(pk *Packet, cb func(rebroadcast bool, err error)) {
	p.mu.Lock()
	handlers := append([]*SimplePeerHandler(nil), p.handlers...)
	p.mu.Unlock()
	for _, h := range handlers {
		if h.mpi == pk.MPI {
			h.rCh <- packetEntry{pk, cb}
		}
	}
}

func (p *SimplePeer) ID() module.PeerID {
	return p.id
}

func (p *SimplePeer) Connect(p2 Peer) *SimplePeer {
	PeerConnect(p, p2)
	return p
}

func (p *SimplePeer) RegisterProto(mpi module.ProtocolInfo) *SimplePeerHandler {
	const chanSize = 1024
	h := &SimplePeerHandler{
		p:   p,
		mpi: mpi,
		rCh: make(chan packetEntry, chanSize),
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers = append(p.handlers, h)
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
	h.p.mu.Lock()
	p := h.p.p
	h.p.mu.Unlock()
	p.notifyPacket(pk, cb)
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
	if !assert.Equal(h.p.t, pi, pe.pk.PI) {
		h.p.t.Logf("data=%s", DumpRLP("  ", pe.pk.Data))
	}
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
	h.p.mu.Lock()
	p := h.p.p
	h.p.mu.Unlock()
	p.notifyPacket(pk, cb)
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
	h.p.mu.Lock()
	p := h.p.p
	h.p.mu.Unlock()
	p.notifyPacket(pk, cb)
}
