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

func (bd *digest) Bytes() []byte {
	if bd.bytes == nil {
		bd.bytes = codec.MustMarshalToBytes(bd.format)
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
	for _, ntd := range bd.format.NetworkTypeDigests {
		for _, nd := range ntd.format.NetworkDigests {
			err := nd.messageList.flush()
			if err != nil {
				return err
			}
		}
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
	if i < len(bd.format.NetworkTypeDigests) && int64(i) == ntid {
		return &bd.format.NetworkTypeDigests[i]
	}
	return nil
}

type networkDigestSlice []networkDigest

func (nds *networkDigestSlice) Len() int {
	return len(*nds)
}

func (nds *networkDigestSlice) Get(i int) []byte {
	return (*nds)[i].NetworkSectionHash()
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

func (ntd *networkTypeDigest) NetworkDigestFor(nid int64) module.NetworkDigest {
	i := sort.Search(
		len(ntd.format.NetworkDigests),
		func(i int) bool {
			return ntd.format.NetworkDigests[i].NetworkID() >= nid
		},
	)
	if i < len(ntd.format.NetworkDigests) && int64(i) == nid {
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
	format      networkDigestFormat
	messageList *messageList
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
	bk, err := dbase.GetBucket(db.ListByMerkleRootFor(mod.UID()))
	if err != nil {
		return nil, err
	}
	bs, err := bk.Get(nd.format.MessagesRoot)
	if err != nil {
		return nil, err
	}
	return newMessageList(bs, dbase, mod), nil
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
	_, err = codec.UnmarshalFromBytes(bs, m)
	if err != nil {
		return nil, err
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

func (l *messageList) Add(msg []byte) {
	m := &message{
		dbase: l.dbase,
		mod:   l.mod,
		data:  msg,
	}
	l.hashesCat.Bytes = append(l.hashesCat.Bytes, m.Hash()...)
	l.messages = append(l.messages, m)
	l.messagesRoot = nil
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
			ntsSlice = append(
				ntsSlice,
				bs.NetworkTypeSectionFor(ntd.NetworkTypeID()),
			)
		}
		bs.networkTypeSections = ntsSlice
	}
	return bs.networkTypeSections
}

func (bs btpSectionFromDigest) NetworkTypeSectionFor(ntid int64) module.NetworkTypeSection {
	if bs.networkTypeSectionFor[ntid] == nil {
		nt, err := bs.view.GetNetworkType(ntid)
		log.Must(err)
		mod := ntm.ForUID(nt.UID)
		npc, err := mod.NewProofContextFromBytes(nt.NextProofContext)
		log.Must(err)
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
	return bs.networkTypeSectionFor[ntid]
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
	if nts.nextProofContext == nil {
		npc, err := nts.mod.NewProofContextFromBytes(nts.nt.NextProofContext)
		log.Must(err)
		nts.nextProofContext = npc
	}
	return nts.nextProofContext
}

func (nts *networkTypeSectionFromDigest) NetworkSectionFor(nid int64) module.NetworkSection {
	nw, err := nts.view.GetNetwork(nid)
	log.Must(err)
	return &networkSectionFromDigest{
		dbase: nts.dbase,
		mod:   nts.mod,
		nw:    nw,
		nd:    nts.ntd.NetworkDigestFor(nid),
	}
}

func (nts *networkTypeSectionFromDigest) NewDecision(height int64, round int32) module.BytesHasher {
	return &networkTypeSectionDecision{
		SrcNetworkID:           []byte(srcNetworkUID),
		DstType:                nts.ntd.NetworkTypeID(),
		Height:                 height,
		Round:                  round,
		NetworkTypeSectionHash: nts.Hash(),
		mod:                    nts.mod,
	}
}

type networkSectionFromDigest struct {
	dbase db.Database
	mod   module.NetworkTypeModule
	nw    *Network
	nd    module.NetworkDigest
}

func (ns *networkSectionFromDigest) Hash() []byte {
	return ns.nw.LastNetworkSectionHash
}

func (ns *networkSectionFromDigest) NetworkID() int64 {
	return ns.nd.NetworkID()
}

func (ns *networkSectionFromDigest) MessageRootNumber() int64 {
	return ns.nw.LastMessagesRootNumber
}

func (ns *networkSectionFromDigest) MessageRootSN() int64 {
	return ns.MessageRootNumber() >> 1
}

func (ns *networkSectionFromDigest) NextProofContextChanged() bool {
	return ns.MessageRootNumber()&0x1 != 0
}

func (ns *networkSectionFromDigest) PrevHash() []byte {
	return ns.nw.PrevNetworkSectionHash
}

func (ns *networkSectionFromDigest) MessageCount() int64 {
	ml, err := ns.nd.MessageList(ns.dbase, ns.mod)
	log.Must(err)
	return ml.Len()
}

func (ns *networkSectionFromDigest) MessagesRoot() []byte {
	return ns.nd.MessagesRoot()
}
