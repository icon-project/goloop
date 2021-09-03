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

import "github.com/icon-project/goloop/module"

type Peer interface {
	ID() module.PeerID
	attach(p Peer)
	detach(p Peer)
	notifyPacket(pk *Packet, cb func(rebroadcast bool, err error))
}

func PeerConnect(p1 Peer, p2 Peer) {
	p1.attach(p2)
	p2.attach(p1)
}

type SendType int

const (
	SendTypeUnicast = SendType(iota)
	SendTypeMulticast
	SendTypeBroadcast
)

type Packet struct {
	SendType SendType
	Src      module.PeerID

	// DstSpec specifies destination.
	// DstSpec is PeerID if SendType is SendTypeUnicast.
	// DstSpec is Role if SendType is SendTypeMulticast.
	// DstSpec is BroadcastType if SendType is SendTypeBroadcast.
	DstSpec interface{}
	MPI     module.ProtocolInfo
	PI      module.ProtocolInfo
	Data    []byte
}

type packetEntry struct {
	pk *Packet
	cb func(bool, error)
}
