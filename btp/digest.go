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
	"sort"

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	hashLen        = 32
	nidFilterBytes = 256 / 8
)

type digestFormat struct {
	NetworkTypeDigests []networkTypeDigest
}

type digest struct {
	format             digestFormat
	bytes              []byte
	hash               []byte
	networkTypeDigests []module.NetworkTypeDigest
	filter             module.BitSetFilter
}

func NewDigestFromBytes(bytes []byte) (module.BTPDigest, error) {
	bd := &digest{}
	_, err := codec.UnmarshalFromBytes(bytes, &bd.format)
	if err != nil {
		return nil, err
	}
	return bd, nil
}

func (bd *digest) Bytes() []byte {
	if bd.bytes == nil {
		bd.bytes = codec.MustMarshalToBytes(&bd.format)
	}
	return bd.bytes
}

func (bd *digest) Hash() []byte {
	if bd.hash == nil {
		crypto.SHA3Sum256(bd.Bytes())
	}
	return bd.hash
}

func (bd *digest) NetworkTypeDigests() []module.NetworkTypeDigest {
	if bd.networkTypeDigests == nil {
		bd.networkTypeDigests = make([]module.NetworkTypeDigest, 0, len(bd.format.NetworkTypeDigests))
		for _, ntd := range bd.format.NetworkTypeDigests {
			bd.networkTypeDigests = append(bd.networkTypeDigests, &ntd)
		}
	}
	return bd.networkTypeDigests
}

func (bd *digest) Flush(dbase db.Database) error {
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return err
	}
	err = bk.Set(bd.Hash(), bd.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (bd *digest) NetworkSectionFilter() module.BitSetFilter {
	if bd.filter == nil {
		bd.filter = module.MakeBitSetFilter(nidFilterBytes)
		for _, ntd := range bd.format.NetworkTypeDigests {
			ntd.updateFilter(bd.filter)
		}
	}
	return bd.filter
}

func (bd *digest) NetworkTypeDigestFor(ntid int64) module.NetworkTypeDigest {
	i := sort.Search(
		len(bd.format.NetworkTypeDigests),
		func(i int) bool {
			return bd.format.NetworkTypeDigests[i].NetworkTypeID() >= ntid
		},
	)
	if i < len(bd.format.NetworkTypeDigests) && bd.format.NetworkTypeDigests[i].NetworkTypeID() == ntid {
		return &bd.format.NetworkTypeDigests[i]
	}
	return nil
}

type networkDigestSlice []networkDigest

func (nds networkDigestSlice) Len() int {
	return len(nds)
}

func (nds networkDigestSlice) Get(i int) []byte {
	return nds[i].NetworkSectionHash()
}

func (nds networkDigestSlice) Search(nid int64) int {
	i := sort.Search(len(nds), func(i int) bool {
		return nds[i].NetworkID() >= nid
	})
	if i < len(nds) && nds[i].NetworkID() == nid {
		return i
	}
	return -1
}

type networkTypeDigestFormat struct {
	NetworkTypeID          int64
	NetworkTypeSectionHash []byte
	NetworkDigests         networkDigestSlice
}

type networkTypeDigest struct {
	format              networkTypeDigestFormat
	networkDigests      []module.NetworkDigest
	networkSectionsRoot []byte
}

func (ntd *networkTypeDigest) NetworkTypeID() int64 {
	return ntd.format.NetworkTypeID
}

func (ntd *networkTypeDigest) NetworkTypeSectionHash() []byte {
	return ntd.format.NetworkTypeSectionHash
}

func (ntd *networkTypeDigest) NetworkDigests() []module.NetworkDigest {
	if ntd.networkDigests == nil {
		ntd.networkDigests = make([]module.NetworkDigest, 0, len(ntd.format.NetworkDigests))
		for _, nd := range ntd.format.NetworkDigests {
			ntd.networkDigests = append(ntd.networkDigests, &nd)
		}
	}
	return ntd.networkDigests
}

func (ntd *networkTypeDigest) NetworkSectionsRootWithMod(mod module.NetworkTypeModule) []byte {
	if ntd.networkSectionsRoot == nil {
		ntd.networkSectionsRoot = mod.MerkleRoot(&ntd.format.NetworkDigests)
	}
	return ntd.networkSectionsRoot
}

func (ntd *networkTypeDigest) NetworkSectionToRootWithMod(mod module.NetworkTypeModule, nid int64) ([]module.MerkleNode, error) {
	i := ntd.format.NetworkDigests.Search(nid)
	if i >= 0 {
		pf := mod.MerkleProof(&ntd.format.NetworkDigests, i)
		return pf, nil
	}
	return nil, errors.Errorf("not found nid=%d", nid)
}

func (ntd *networkTypeDigest) NetworkDigestFor(nid int64) module.NetworkDigest {
	i := ntd.format.NetworkDigests.Search(nid)
	if i >= 0 {
		return &ntd.format.NetworkDigests[i]
	}
	return nil
}

func (ntd *networkTypeDigest) updateFilter(f module.BitSetFilter) {
	for _, nd := range ntd.networkDigests {
		f.Set(nd.NetworkID())
	}
}

func (ntd *networkTypeDigest) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&ntd.format)
}

func (ntd *networkTypeDigest) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&ntd.format)
}

type networkDigestFormat struct {
	NetworkID          int64
	NetworkSectionHash []byte
	MessagesRoot       []byte
}

type networkDigest struct {
	format        networkDigestFormat
	messageHashes []byte
}

func (nd *networkDigest) NetworkID() int64 {
	return nd.format.NetworkID
}

func (nd *networkDigest) NetworkSectionHash() []byte {
	return nd.format.NetworkSectionHash
}

func (nd *networkDigest) MessagesRoot() []byte {
	return nd.format.MessagesRoot
}

func (nd *networkDigest) MessageList(
	dbase db.Database,
	mod module.NetworkTypeModule,
) (module.BTPMessageList, error) {
	if nd.messageHashes == nil {
		bk, err := dbase.GetBucket(db.ListByMerkleRootFor(mod.UID()))
		if err != nil {
			return nil, err
		}
		bs, err := bk.Get(nd.format.MessagesRoot)
		if err != nil {
			return nil, err
		}
		nd.messageHashes = bs
	}
	return newMessageList(nd.messageHashes, dbase, mod), nil
}

func (nd *networkDigest) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&nd.format)
}

func (nd *networkDigest) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&nd.format)
}

type hashesCat struct {
	Bytes []byte
}

func makeHashesCat(c int) hashesCat {
	return hashesCat{
		Bytes: make([]byte, 0, c),
	}
}

func (hc *hashesCat) Append(hash []byte) {
	hc.Bytes = append(hc.Bytes, hash...)
}

func (hc *hashesCat) Len() int {
	return len(hc.Bytes) / hashLen
}

func (hc *hashesCat) Get(i int) []byte {
	return hc.Bytes[i*hashLen : (i+1)*hashLen]
}

type messageList struct {
	hashesCat
	dbase        db.Database
	mod          module.NetworkTypeModule
	messages     []*message
	messagesRoot []byte
}

func newMessageList(
	messageHashes []byte,
	dbase db.Database,
	mod module.NetworkTypeModule,
) *messageList {
	l := &messageList{
		hashesCat: hashesCat{
			Bytes: messageHashes,
		},
		dbase:    dbase,
		mod:      mod,
		messages: make([]*message, len(messageHashes)/hashLen),
	}
	return l
}

func (l *messageList) Bytes() []byte {
	return l.hashesCat.Bytes
}

func (l *messageList) MessagesRoot() []byte {
	if l.messagesRoot == nil {
		l.messagesRoot = l.mod.MerkleRoot(&l.hashesCat)
	}
	return l.messagesRoot
}

func (l *messageList) Get(idx int) (module.BTPMessage, error) {
	if l.messages[idx] != nil {
		return l.messages[idx], nil
	}
	bk, err := l.dbase.GetBucket(db.BytesByHashFor(l.mod.UID()))
	if err != nil {
		return nil, err
	}
	msgHash := l.hashesCat.Get(idx)
	bs, err := bk.Get(msgHash)
	if err != nil {
		return nil, err
	}
	m := &message{
		dbase: l.dbase,
		mod:   l.mod,
		data:  bs,
		hash:  msgHash,
	}
	l.messages[idx] = m
	return m, nil
}

func (l *messageList) flush() error {
	bk, err := l.dbase.GetBucket(db.ListByMerkleRootFor(l.mod.UID()))
	if err != nil {
		return err
	}
	err = bk.Set(l.MessagesRoot(), l.Bytes())
	if err != nil {
		return err
	}
	for _, m := range l.messages {
		err = m.flush()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *messageList) Len() int64 {
	return int64(l.hashesCat.Len())
}

type message struct {
	dbase db.Database
	mod   module.NetworkTypeModule
	data  []byte
	hash  []byte
}

func (m *message) Hash() []byte {
	if m.hash == nil {
		m.hash = m.mod.Hash(m.data)
	}
	return m.hash
}

func (m *message) Bytes() []byte {
	return m.data
}

func (m *message) flush() error {
	bk, err := m.dbase.GetBucket(db.BytesByHashFor(m.mod.UID()))
	if err != nil {
		return err
	}
	return bk.Set(m.Hash(), m.Bytes())
}

// NewSection returns a new Section. view shall have the final value for a
// transition.
func NewSection(
	digest module.BTPDigest,
	view StateView,
	dbase db.Database,
) (module.BTPSection, error) {
	return &btpSectionFromDigest{
		digest:                digest,
		view:                  view,
		dbase:                 dbase,
		networkTypeSectionFor: make(map[int64]*networkTypeSectionFromDigest),
	}, nil
}

type btpSectionFromDigest struct {
	digest                module.BTPDigest
	view                  StateView
	dbase                 db.Database
	networkTypeSectionFor map[int64]*networkTypeSectionFromDigest
	networkTypeSections   []module.NetworkTypeSection
}

func (bs btpSectionFromDigest) Digest() module.BTPDigest {
	return bs.digest
}

func (bs btpSectionFromDigest) NetworkTypeSections() []module.NetworkTypeSection {
	if bs.networkTypeSections == nil {
		ntdSlice := bs.digest.NetworkTypeDigests()
		ntsSlice := make([]module.NetworkTypeSection, 0, len(ntdSlice))
		for _, ntd := range ntdSlice {
			nts, err := bs.NetworkTypeSectionFor(ntd.NetworkTypeID())
			log.Must(err)
			ntsSlice = append(ntsSlice, nts)
		}
		bs.networkTypeSections = ntsSlice
	}
	return bs.networkTypeSections
}

func (bs btpSectionFromDigest) NetworkTypeSectionFor(ntid int64) (module.NetworkTypeSection, error) {
	if bs.networkTypeSectionFor[ntid] == nil {
		nt, err := bs.view.GetNetworkType(ntid)
		if err != nil {
			return nil, err
		}
		mod := ntm.ForUID(nt.UID)
		npc, err := mod.NewProofContextFromBytes(nt.NextProofContext)
		if err != nil {
			return nil, err
		}
		nts := &networkTypeSectionFromDigest{
			view:             bs.view,
			dbase:            bs.dbase,
			mod:              mod,
			nt:               nt,
			ntd:              bs.digest.NetworkTypeDigestFor(ntid),
			nextProofContext: npc,
		}
		bs.networkTypeSectionFor[ntid] = nts
	}
	return bs.networkTypeSectionFor[ntid], nil
}

type networkTypeSectionFromDigest struct {
	view             StateView
	dbase            db.Database
	mod              module.NetworkTypeModule
	nt               *NetworkType
	ntd              module.NetworkTypeDigest
	nextProofContext module.BTPProofContext
}

func (nts *networkTypeSectionFromDigest) NetworkTypeID() int64 {
	return nts.ntd.NetworkTypeID()
}

func (nts *networkTypeSectionFromDigest) Hash() []byte {
	return nts.ntd.NetworkTypeSectionHash()
}

func (nts *networkTypeSectionFromDigest) NetworkSectionsRoot() []byte {
	return nts.ntd.NetworkSectionsRootWithMod(nts.mod)
}

func (nts *networkTypeSectionFromDigest) NextProofContext() module.BTPProofContext {
	return nts.nextProofContext
}

func (nts *networkTypeSectionFromDigest) NetworkSectionFor(nid int64) (module.NetworkSection, error) {
	nw, err := nts.view.GetNetwork(nid)
	if err != nil {
		return nil, err
	}
	ns, err := newNetworkSectionFromDigest(
		nts.dbase,
		nts.mod,
		nw,
		nts.ntd.NetworkDigestFor(nid),
	)
	if err != nil {
		return nil, err
	}
	return ns, nil
}

func (nts *networkTypeSectionFromDigest) NewDecision(
	srcNetworkUID []byte,
	height int64,
	round int32,
) module.BytesHasher {
	return &networkTypeSectionDecision{
		SrcNetworkID:           srcNetworkUID,
		DstType:                nts.ntd.NetworkTypeID(),
		Height:                 height,
		Round:                  round,
		NetworkTypeSectionHash: nts.Hash(),
		mod:                    nts.mod,
	}
}

func (nts *networkTypeSectionFromDigest) NetworkSectionToRoot(nid int64) ([]module.MerkleNode, error) {
	return nts.ntd.NetworkSectionToRootWithMod(nts.mod, nid)
}

type networkSectionFromDigest struct {
	dbase        db.Database
	mod          module.NetworkTypeModule
	nw           *Network
	nd           module.NetworkDigest
	updateNumber int64
	messageCount int64
}

func newNetworkSectionFromDigest(
	dbase db.Database,
	mod module.NetworkTypeModule,
	nw *Network,
	nd module.NetworkDigest,
) (*networkSectionFromDigest, error) {
	ml, err := nd.MessageList(dbase, mod)
	if err != nil {
		return nil, err
	}
	updateNumber := (nw.NextMessageSN - ml.Len()) << 1
	if nw.NextProofContextChanged {
		updateNumber |= 1
	}
	return &networkSectionFromDigest{
		dbase:        dbase,
		mod:          mod,
		nw:           nw,
		nd:           nd,
		updateNumber: updateNumber,
		messageCount: ml.Len(),
	}, nil
}

func (ns *networkSectionFromDigest) Hash() []byte {
	return ns.nw.LastNetworkSectionHash
}

func (ns *networkSectionFromDigest) NetworkID() int64 {
	return ns.nd.NetworkID()
}

func (ns *networkSectionFromDigest) UpdateNumber() int64 {
	return ns.updateNumber
}

func (ns *networkSectionFromDigest) FirstMessageSN() int64 {
	return ns.nw.NextMessageSN - ns.MessageCount()
}

func (ns *networkSectionFromDigest) NextProofContextChanged() bool {
	return ns.nw.NextProofContextChanged
}

func (ns *networkSectionFromDigest) PrevHash() []byte {
	return ns.nw.PrevNetworkSectionHash
}

func (ns *networkSectionFromDigest) MessageCount() int64 {
	return ns.messageCount
}

func (ns *networkSectionFromDigest) MessagesRoot() []byte {
	return ns.nd.MessagesRoot()
}
