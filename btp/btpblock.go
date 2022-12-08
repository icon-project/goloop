/*
 * Copyright 2022 ICON Foundation
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

package btp

import (
	"github.com/icon-project/goloop/common/atomic"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

type btpBlockHeaderFormat struct {
	MainHeight             int64
	Round                  int32
	NextProofContextHash   []byte
	NetworkSectionToRoot   []module.MerkleNode
	NetworkID              int64
	UpdateNumber           int64
	PrevNetworkSectionHash []byte
	MessageCount           int64
	MessagesRoot           []byte
	NextProofContext       []byte
}

type btpBlockHeader struct {
	format btpBlockHeaderFormat
	bytes  atomic.Cache[[]byte]
}

func (bh *btpBlockHeader) MainHeight() int64 {
	return bh.format.MainHeight
}

func (bh *btpBlockHeader) Round() int32 {
	return bh.format.Round
}

func (bh *btpBlockHeader) NextProofContextHash() []byte {
	return bh.format.NextProofContextHash
}

func (bh *btpBlockHeader) NetworkSectionToRoot() []module.MerkleNode {
	return bh.format.NetworkSectionToRoot
}

func (bh *btpBlockHeader) NetworkID() int64 {
	return bh.format.NetworkID
}

func (bh *btpBlockHeader) UpdateNumber() int64 {
	return bh.format.UpdateNumber
}

func (bh *btpBlockHeader) FirstMessageSN() int64 {
	return bh.format.UpdateNumber >> 1
}

func (bh *btpBlockHeader) NextProofContextChanged() bool {
	return bh.format.UpdateNumber&0x1 != 0
}

func (bh *btpBlockHeader) PrevNetworkSectionHash() []byte {
	return bh.format.PrevNetworkSectionHash
}

func (bh *btpBlockHeader) MessageCount() int64 {
	return bh.format.MessageCount
}

func (bh *btpBlockHeader) MessagesRoot() []byte {
	return bh.format.MessagesRoot
}

func (bh *btpBlockHeader) NextProofContext() []byte {
	return bh.format.NextProofContext
}

func (bh *btpBlockHeader) HeaderBytes() []byte {
	return bh.bytes.Get(func() []byte {
		return codec.MustMarshalToBytes(&bh.format)
	})
}

// NewBTPBlockHeader returns a new BTPBlockHeader for the height and nid. If flag's
// IncludeNextProofContext bit is on, the header includes NextProofContext.
func NewBTPBlockHeader(
	height int64,
	round int32,
	nts module.NetworkTypeSection,
	nid int64,
	flag uint,
) (module.BTPBlockHeader, error) {
	bb := &btpBlockHeader{}
	bb.format.MainHeight = height
	bb.format.Round = round
	bb.format.NextProofContextHash = nts.NextProofContext().Hash()
	pf, err := nts.NetworkSectionToRoot(nid)
	if err != nil {
		return nil, err
	}
	bb.format.NetworkSectionToRoot = pf
	bb.format.NetworkID = nid
	ns, err := nts.NetworkSectionFor(nid)
	if err != nil {
		return nil, err
	}
	bb.format.UpdateNumber = ns.UpdateNumber()
	bb.format.PrevNetworkSectionHash = ns.PrevHash()
	bb.format.MessageCount = ns.MessageCount()
	bb.format.MessagesRoot = ns.MessagesRoot()
	if flag&module.FlagNextProofContext != 0 || ns.NextProofContextChanged() {
		bb.format.NextProofContext = nts.NextProofContext().Bytes()
	}
	return bb, nil
}
