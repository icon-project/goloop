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
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type btpSection struct {
	networkTypeSections networkTypeSectionSlice
	digest              *btpSectionDigest
}

var ZeroBTPSection = newBTPSection(nil)

func newBTPSection(ntsSlice networkTypeSectionSlice) *btpSection {
	bs := &btpSection{
		networkTypeSections: ntsSlice,
	}
	bs.digest = &btpSectionDigest{
		bs: bs,
	}
	return bs
}

func (bs *btpSection) Digest() module.BTPDigest {
	return bs.digest
}

func (bs *btpSection) NetworkTypeSections() []module.NetworkTypeSection {
	return bs.networkTypeSections
}

func (bs *btpSection) NetworkTypeSectionFor(ntid int64) (module.NetworkTypeSection, error) {
	nts := bs.networkTypeSections.Search(ntid)
	if nts == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found ntid=%d", ntid)
	}
	return nts, nil
}

type btpSectionDigest struct {
	bs                 *btpSection
	bytes              []byte
	hash               []byte
	networkTypeDigests []module.NetworkTypeDigest
	filter             module.BitSetFilter
}

func (bsd *btpSectionDigest) Bytes() []byte {
	if bsd.bytes == nil {
		if len(bsd.bs.networkTypeSections) == 0 {
			bsd.bytes = make([]byte, 0)
		} else {
			e := codec.NewEncoderBytes(&bsd.bytes)
			if len(bsd.bs.networkTypeSections) > 0 {
				e2, _ := e.EncodeList()  // bd struct
				e3, _ := e2.EncodeList() // ntd slice
				for _, nts := range bsd.bs.networkTypeSections {
					_ = nts.(*networkTypeSection).encodeDigest(e3)
				}
			}
			_ = e.Close()
		}
	}
	if len(bsd.bytes) == 0 {
		return nil
	}
	return bsd.bytes
}

func (bsd *btpSectionDigest) Hash() []byte {
	if bsd.hash == nil {
		if bsd.Bytes() == nil {
			bsd.hash = make([]byte, 0)
		} else {
			bsd.hash = crypto.SHA3Sum256(bsd.Bytes())
		}
	}
	if len(bsd.hash) == 0 {
		return nil
	}
	return bsd.hash
}

func (bsd *btpSectionDigest) NetworkTypeDigests() []module.NetworkTypeDigest {
	if bsd.networkTypeDigests == nil {
		bsd.networkTypeDigests = make([]module.NetworkTypeDigest, 0, len(bsd.bs.networkTypeSections))
		for _, ntd := range bsd.bs.networkTypeSections {
			bsd.networkTypeDigests = append(bsd.networkTypeDigests, ntd.(*networkTypeSection))
		}
	}
	return bsd.networkTypeDigests
}

func (bsd *btpSectionDigest) NetworkTypeDigestFor(ntid int64) module.NetworkTypeDigest {
	return bsd.bs.networkTypeSections.Search(ntid).(*networkTypeSection)
}

func (bsd *btpSectionDigest) Flush(dbase db.Database) error {
	if bsd.Hash() == nil {
		return nil
	}
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return err
	}
	err = bk.Set(bsd.Hash(), bsd.Bytes())
	if err != nil {
		return err
	}
	for _, nts := range bsd.bs.networkTypeSections {
		err = nts.(*networkTypeSection).flushMessages(dbase)
		if err != nil {
			return err
		}
	}
	return nil
}

func (bsd *btpSectionDigest) NetworkSectionFilter() module.BitSetFilter {
	if bsd.filter.Bytes() == nil {
		bsd.filter = module.MakeBitSetFilter(NSFilterCap)
		for _, nts := range bsd.bs.networkTypeSections {
			nts.(*networkTypeSection).updateFilter(bsd.filter)
		}
	}
	return bsd.filter
}

type networkTypeSection struct {
	networkTypeID        int64
	nextProofContext     module.BTPProofContext
	nextProofContextHash []byte
	networkSections      networkSectionSlice
	networkSectionsRoot  []byte
	networkDigests       []module.NetworkDigest
	mod                  module.NetworkTypeModule
	npcChanged           bool
	hash                 []byte
}

func newNetworkTypeSection(
	ntid int64,
	nt NetworkTypeView,
	nsSlice networkSectionSlice,
	npcChanged bool,
) (*networkTypeSection, error) {
	mod := ntm.ForUID(nt.UID())
	npc, err := mod.NewProofContextFromBytes(nt.NextProofContext())
	if err != nil {
		return nil, err
	}
	nts := &networkTypeSection{
		networkTypeID:       ntid,
		nextProofContext:    npc,
		networkSections:     nsSlice,
		networkSectionsRoot: mod.MerkleRoot(&nsSlice),
		mod:                 mod,
		npcChanged:          npcChanged,
	}
	ntsFormat := nts.networkTypeSectionFormat()
	nts.hash = mod.Hash(codec.MustMarshalToBytes(&ntsFormat))
	return nts, nil
}

type networkTypeSectionFormat struct {
	NextProofContextHash []byte
	NetworkSectionsRoot  []byte
}

func (nts *networkTypeSection) networkTypeSectionFormat() networkTypeSectionFormat {
	return networkTypeSectionFormat{
		NextProofContextHash: nts.nextProofContext.Hash(),
		NetworkSectionsRoot:  nts.networkSectionsRoot,
	}
}

func (nts *networkTypeSection) NetworkTypeID() int64 {
	return nts.networkTypeID
}

func (nts *networkTypeSection) Hash() []byte {
	return nts.hash
}

func (nts *networkTypeSection) NetworkSectionsRoot() []byte {
	return nts.networkSectionsRoot
}

func (nts *networkTypeSection) NetworkSectionToRoot(nid int64) ([]module.MerkleNode, error) {
	return nts.NetworkSectionToRootWithMod(nts.mod, nid)
}

func (nts *networkTypeSection) NextProofContext() module.BTPProofContext {
	return nts.nextProofContext
}

func (nts *networkTypeSection) NetworkSections() []module.NetworkSection {
	return nts.networkSections
}

func (nts *networkTypeSection) NetworkTypeSectionHash() []byte {
	return nts.hash
}

// nextProofContextChanged returns true if NS.NextProofContextChanged() is true
// for any child NS of this NTS.
func (nts *networkTypeSection) nextProofContextChanged() bool {
	return nts.npcChanged
}

func (nts *networkTypeSection) NetworkDigests() []module.NetworkDigest {
	if nts.networkDigests == nil {
		nts.networkDigests = make([]module.NetworkDigest, 0, len(nts.networkSections))
		for _, ns := range nts.networkSections {
			nts.networkDigests = append(nts.networkDigests, ns.(*networkSection))
		}
	}
	return nts.networkDigests
}

func (nts *networkTypeSection) NetworkDigestFor(nid int64) module.NetworkDigest {
	ns, _ := nts.networkSections.Search(nid)
	if ns != nil {
		return ns.(*networkSection)
	}
	return nil
}

func (nts *networkTypeSection) NetworkSectionsRootWithMod(mod module.NetworkTypeModule) []byte {
	if nts.mod == mod {
		return nts.networkSectionsRoot
	}
	return mod.MerkleRoot(nts.networkSections)
}

func (nts *networkTypeSection) NetworkSectionToRootWithMod(mod module.NetworkTypeModule, nid int64) ([]module.MerkleNode, error) {
	_, i := nts.networkSections.Search(nid)
	if i < 0 {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found nid=%d", nid)
	}
	return mod.MerkleProof(nts.networkSections, i), nil
}

func (nts *networkTypeSection) NetworkSectionFor(nid int64) (module.NetworkSection, error) {
	ns, _ := nts.networkSections.Search(nid)
	if ns == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "not found nid=%d", nid)
	}
	return ns, nil
}

type networkTypeSectionDecision struct {
	SrcNetworkID           []byte
	DstType                int64
	Height                 int64
	Round                  int32
	NetworkTypeSectionHash []byte
	mod                    module.NetworkTypeModule
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

func (nts *networkTypeSection) NewDecision(
	srcNetworkUID []byte,
	height int64,
	round int32,
) module.BytesHasher {
	return &networkTypeSectionDecision{
		SrcNetworkID:           srcNetworkUID,
		DstType:                nts.networkTypeID,
		Height:                 height,
		Round:                  round,
		NetworkTypeSectionHash: nts.hash,
		mod:                    nts.mod,
	}
}

func (nts *networkTypeSection) updateFilter(f module.BitSetFilter) {
	for _, ns := range nts.networkSections {
		f.Set(ns.NetworkID())
	}
}

func (nts *networkTypeSection) flushMessages(dbase db.Database) error {
	for _, ns := range nts.networkSections {
		err := ns.(*networkSection).flushMessages(dbase)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nts *networkTypeSection) encodeDigest(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	err = e2.EncodeMulti(
		nts.NetworkTypeID(),
		nts.NetworkTypeSectionHash(),
	)
	if err != nil {
		return err
	}
	e3, err := e2.EncodeList() // nd slice
	if err != nil {
		return err
	}
	for _, ns := range nts.networkSections {
		err = ns.(*networkSection).encodeDigest(e3)
		if err != nil {
			return err
		}
	}
	return nil
}

type networkSection struct {
	networkID     int64
	updateNumber  int64
	prevHash      []byte
	messages      [][]byte
	messageHashes hashesCat
	messagesRoot  []byte
	mod           module.NetworkTypeModule
	hash          []byte
}

func newNetworkSection(
	nid int64,
	nw NetworkView,
	ne *networkEntry,
	mod module.NetworkTypeModule,
) *networkSection {
	updateNumber := (nw.NextMessageSN() - int64(len(ne.messages))) << 1
	if nw.NextProofContextChanged() {
		updateNumber |= 1
	}
	ns := &networkSection{
		networkID:    nid,
		updateNumber: updateNumber,
		prevHash:     nw.LastNetworkSectionHash(),
		messages:     ne.messages,
	}
	ns.messageHashes = makeHashesCat(len(ne.messages))
	for _, msg := range ne.messages {
		ns.messageHashes.Append(mod.Hash(msg))
	}
	ns.messagesRoot = mod.MerkleRoot(&ns.messageHashes)
	ns.mod = mod
	nsFormat := ns.networkSectionFormat()
	ns.hash = mod.Hash(codec.MustMarshalToBytes(&nsFormat))
	return ns
}

type networkSectionFormat struct {
	NetworkID    int64
	UpdateNumber int64
	PrevHash     []byte
	MessageCount int64
	MessagesRoot []byte
}

func (ns *networkSection) networkSectionFormat() networkSectionFormat {
	return networkSectionFormat{
		NetworkID:    ns.networkID,
		UpdateNumber: ns.updateNumber,
		PrevHash:     ns.prevHash,
		MessageCount: int64(ns.messageHashes.Len()),
		MessagesRoot: ns.messagesRoot,
	}
}

func (ns *networkSection) NetworkID() int64 {
	return ns.networkID
}

func (ns *networkSection) UpdateNumber() int64 {
	return ns.updateNumber
}

func (ns *networkSection) FirstMessageSN() int64 {
	return ns.updateNumber >> 1
}

func (ns *networkSection) NextProofContextChanged() bool {
	return ns.updateNumber&1 != 0
}

func (ns *networkSection) PrevHash() []byte {
	return ns.prevHash
}

func (ns *networkSection) MessageCount() int64 {
	return int64(ns.messageHashes.Len())
}

func (ns *networkSection) MessagesRoot() []byte {
	return ns.messagesRoot
}

func (ns *networkSection) Hash() []byte {
	return ns.hash
}

func (ns *networkSection) NetworkSectionHash() []byte {
	return ns.hash
}

func (ns *networkSection) MessageList(dbase db.Database, mod module.NetworkTypeModule) (module.BTPMessageList, error) {
	bk, err := dbase.GetBucket(mod.ListByMerkleRootBucket())
	if err != nil {
		return nil, err
	}
	bs, err := bk.Get(ns.messagesRoot)
	if err != nil {
		return nil, err
	}
	return newMessageList(bs, dbase, ns.mod), nil
}

func (ns *networkSection) flushMessages(dbase db.Database) error {
	if ns.messagesRoot == nil {
		return nil
	}
	bk, err := dbase.GetBucket(ns.mod.ListByMerkleRootBucket())
	if err != nil {
		return err
	}
	err = bk.Set(ns.messagesRoot, ns.messageHashes.Bytes)
	if err != nil {
		return err
	}
	bk, err = dbase.GetBucket(ns.mod.BytesByHashBucket())
	for i, msg := range ns.messages {
		err = bk.Set(ns.messageHashes.Get(i), msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ns *networkSection) encodeDigest(e codec.Encoder) error {
	return e.EncodeListOf(
		ns.NetworkID(),
		ns.NetworkSectionHash(),
		ns.MessagesRoot(),
	)
}
