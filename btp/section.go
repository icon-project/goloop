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
	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const srcNetworkUID = "icon"

type btpSection struct {
	networkTypeSections []module.NetworkTypeSection
}

func (bs *btpSection) Digest(dbase db.Database) module.BTPDigest {
	networkTypeDigests := make([]*networkTypeDigest, 0, len(bs.networkTypeSections))
	for _, nts := range bs.networkTypeSections {
		ntd := nts.(*networkTypeSection).digest(dbase)
		networkTypeDigests = append(networkTypeDigests, ntd)
	}
	return &digest{
		digestFormat: digestFormat{
			NetworkTypeDigests: networkTypeDigests,
		},
		dbase: dbase,
	}
}

func (bs *btpSection) NetworkTypeSections() []module.NetworkTypeSection {
	return bs.networkTypeSections
}

type networkTypeSectionFormat struct {
	NextProofContextHash []byte
	NetworkSectionsRoot  []byte
}

type networkTypeSection struct {
	format           networkTypeSectionFormat
	networkTypeID    int64
	nextProofContext module.BTPProofContext
	networkSections  []module.NetworkSection
	hash             []byte
	mod              ntm.Module
}

func newNetworkTypeSection(
	ntid int64,
	nt *NetworkType,
	nsSlice []module.NetworkSection,
) *networkTypeSection {
	nts := &networkTypeSection{}
	nts.format.NextProofContextHash = nt.NextProofContextHash
	hashes := make([][]byte, 0, len(nts.networkSections))
	for _, ns := range nts.networkSections {
		hashes = append(hashes, ns.Hash())
	}
	mod := ntm.ForUID(nt.UID)
	nts.format.NetworkSectionsRoot = mod.MerkleRoot(hashes)
	nts.networkTypeID = ntid
	nts.networkSections = nsSlice
	nts.hash = mod.Hash(codec.MustMarshalToBytes(&nts.format))
	nts.mod = mod
	return nts
}

func (nts *networkTypeSection) NetworkTypeID() int64 {
	return nts.networkTypeID
}

func (nts *networkTypeSection) Hash() []byte {
	return nts.hash
}

func (nts *networkTypeSection) NetworkSectionsRoot() []byte {
	return nts.format.NetworkSectionsRoot
}

func (nts *networkTypeSection) NextProofContext() module.BTPProofContext {
	return nts.nextProofContext
}

func (nts *networkTypeSection) NetworkSections() []module.NetworkSection {
	return nts.networkSections
}

func (nts *networkTypeSection) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&nts.format)
}

func (nts *networkTypeSection) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&nts.format)
}

func (nts *networkTypeSection) digest(dbase db.Database) *networkTypeDigest {
	ndSlice := make([]*networkDigest, 0, len(nts.networkSections))
	for _, ns := range nts.networkSections {
		ndSlice = append(ndSlice, ns.(*networkSection).digest(nts.mod, dbase))
	}
	ntd := &networkTypeDigest{
		format: networkTypeDigestFormat{
			NetworkTypeID:          nts.NetworkTypeID(),
			NetworkTypeSectionHash: nts.hash,
			NetworkDigests:         make([]networkDigest, 0, len(nts.networkSections)),
		},
		mod:   nts.mod,
		dbase: dbase,
	}
	return ntd
}

type networkTypeSectionDecision struct {
	SrcNetworkID           []byte
	DstType                int64
	Height                 int64
	Round                  int32
	NetworkTypeSectionHash []byte
	mod                    ntm.Module
	bytes                  []byte
	hash                   []byte
}

func (d *networkTypeSectionDecision) Bytes() []byte {
	if d.bytes == nil {
		d.bytes = codec.MustMarshalToBytes(d)
	}
	return d.bytes
}

func (d *networkTypeSectionDecision) Hash() []byte {
	if d.hash == nil {
		d.hash = d.mod.Hash(d.Bytes())
	}
	return d.hash
}

func (nts *networkTypeSection) NewDecision(height int64, round int32) module.BytesHasher {
	return &networkTypeSectionDecision{
		SrcNetworkID:           []byte(srcNetworkUID),
		DstType:                nts.networkTypeID,
		Height:                 height,
		Round:                  round,
		NetworkTypeSectionHash: nts.hash,
		mod:                    nts.mod,
	}
}

type networkSectionFormat struct {
	NetworkID         int64
	MessageRootNumber int64
	PrevHash          []byte
	MessageCount      int64
	MessagesRoot      []byte
}

type networkSection struct {
	format networkSectionFormat
	hash   []byte
}

func newNetworkSection(
	nid int64,
	nw *Network,
	ne *networkEntry,
	mod ntm.Module,
) *networkSection {
	ns := &networkSection{}
	ns.format.NetworkID = nid
	ns.format.MessageRootNumber = nw.LastMessagesRootNumber
	ns.format.PrevHash = nw.LastNetworkSectionHash
	ns.format.MessageCount = int64(len(ne.messages))
	hashes := make([][]byte, 0, len(ne.messages))
	for _, msg := range ne.messages {
		hashes = append(hashes, mod.Hash(msg))
	}
	ns.format.MessagesRoot = mod.MerkleRoot(hashes)
	ns.hash = mod.Hash(codec.MustMarshalToBytes(&ns.format))
	return ns
}

func (ns *networkSection) NetworkID() int64 {
	return ns.format.NetworkID
}

func (ns *networkSection) MessageRootNumber() int64 {
	return ns.format.MessageRootNumber
}

func (ns *networkSection) MessageRootSN() int64 {
	return ns.format.MessageRootNumber >> 1
}

func (ns *networkSection) NextProofContextChanged() bool {
	return ns.format.MessageRootNumber&1 != 0
}

func (ns *networkSection) PrevHash() []byte {
	return ns.format.PrevHash
}

func (ns *networkSection) MessageCount() int64 {
	return ns.format.MessageCount
}

func (ns *networkSection) MessagesRoot() []byte {
	return ns.format.MessagesRoot
}

func (ns *networkSection) Hash() []byte {
	return ns.hash
}

func (ns *networkSection) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&ns.format)
}

func (ns *networkSection) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&ns.format)
}

func (ns *networkSection) digest(
	mod ntm.Module,
	dbase db.Database,
) *networkDigest {
	nd := &networkDigest{}
	nd.format.NetworkID = ns.NetworkID()
	nd.format.NetworkSectionHash = ns.Hash()
	nd.format.MessagesRoot = ns.MessagesRoot()
	nd.mod = mod
	nd.dbase = dbase
	return nd
}

// NewSection returns a new Section. view shall have the final value for a
// transition.
func NewSection(view StateView, digest module.BTPDigest) (module.BTPSection, error) {
	return nil, nil
}
