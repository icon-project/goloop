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
	"io"
	"sort"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	hashLen     = 32
	NSFilterCap = 256 / 8
)

type digestCore interface {
	Bytes() []byte
	Hash() []byte
	NetworkTypeDigests() []module.NetworkTypeDigest
	Flush(dbase db.Database) error
}

type digest struct {
	digestCore
	hash                   []byte
	filter                 module.BitSetFilter
	ntsHashEntryListFormat []module.NTSHashEntryFormat
}

func (bd *digest) Hash() []byte {
	if bd.hash == nil {
		if bd.Bytes() == nil {
			bd.hash = []byte{}
		} else {
			bd.hash = crypto.SHA3Sum256(bd.Bytes())
		}
	}
	if len(bd.hash) == 0 {
		return nil
	}
	return bd.hash
}

func (bd *digest) NetworkTypeDigestFor(ntid int64) module.NetworkTypeDigest {
	ntdSlice := bd.NetworkTypeDigests()
	i := sort.Search(
		len(ntdSlice),
		func(i int) bool {
			return ntdSlice[i].NetworkTypeID() >= ntid
		},
	)
	if i < len(ntdSlice) && ntdSlice[i].NetworkTypeID() == ntid {
		return ntdSlice[i]
	}
	return nil
}

func (bd *digest) NetworkTypeIDFromNID(nid int64) (int64, error) {
	for _, ntd := range bd.NetworkTypeDigests() {
		for _, nd := range ntd.NetworkDigests() {
			if nd.NetworkID() == nid {
				return ntd.NetworkTypeID(), nil
			}
		}
	}
	return 0, errors.Wrapf(errors.ErrNotFound, "not found nid=%d", nid)
}

func (bd *digest) NetworkSectionFilter() module.BitSetFilter {
	if bd.filter.Bytes() == nil {
		bd.filter = module.MakeBitSetFilter(NSFilterCap)
		for _, ntd := range bd.NetworkTypeDigests() {
			for _, nd := range ntd.NetworkDigests() {
				bd.filter.Set(nd.NetworkID())
			}
		}
	}
	return bd.filter
}

func (bd *digest) NTSHashEntryListFormat() []module.NTSHashEntryFormat {
	if bd.ntsHashEntryListFormat == nil {
		ntdSlice := bd.NetworkTypeDigests()
		ntsHashEntries := make([]module.NTSHashEntryFormat, 0, len(ntdSlice))
		for _, ntd := range ntdSlice {
			ntsHashEntries = append(ntsHashEntries, module.NTSHashEntryFormat{
				NetworkTypeID:          ntd.NetworkTypeID(),
				NetworkTypeSectionHash: ntd.NetworkTypeSectionHash(),
			})
		}
	}
	return bd.ntsHashEntryListFormat
}

func (bd *digest) NTSHashEntryCount() int {
	return len(bd.NetworkTypeDigests())
}

func (bd *digest) NTSHashEntryAt(i int) module.NTSHashEntryFormat {
	ntd := bd.NetworkTypeDigests()[i]
	return module.NTSHashEntryFormat{
		NetworkTypeID:          ntd.NetworkTypeID(),
		NetworkTypeSectionHash: ntd.NetworkTypeSectionHash(),
	}
}

type networkTypeDigestCore interface {
	NetworkTypeID() int64
	NetworkTypeSectionHash() []byte
	NetworkDigests() []module.NetworkDigest
}

type networkTypeDigest struct {
	networkTypeDigestCore
	networkSectionsRoot []byte
}

func (ntd *networkTypeDigest) NetworkDigestFor(nid int64) module.NetworkDigest {
	ndSlice := networkDigestSlice(ntd.NetworkDigests())
	i := ndSlice.Search(nid)
	if i >= 0 {
		return ndSlice[i]
	}
	return nil
}

func (ntd *networkTypeDigest) NetworkSectionsRootWithMod(mod module.NetworkTypeModule) []byte {
	if ntd.networkSectionsRoot == nil {
		ndSlice := networkDigestSlice(ntd.NetworkDigests())
		ntd.networkSectionsRoot = mod.MerkleRoot(&ndSlice)
	}
	return ntd.networkSectionsRoot
}

func (ntd *networkTypeDigest) NetworkSectionToRootWithMod(mod module.NetworkTypeModule, nid int64) ([]module.MerkleNode, error) {
	ndSlice := networkDigestSlice(ntd.NetworkDigests())
	i := ndSlice.Search(nid)
	if i >= 0 {
		pf := mod.MerkleProof(&ndSlice, i)
		return pf, nil
	}
	return nil, errors.Errorf("not found nid=%d", nid)
}

type networkDigestSlice []module.NetworkDigest

func (nds *networkDigestSlice) Len() int {
	return len(*nds)
}

func (nds *networkDigestSlice) Get(i int) []byte {
	return (*nds)[i].NetworkSectionHash()
}

func (nds *networkDigestSlice) Search(nid int64) int {
	i := sort.Search(len(*nds), func(i int) bool {
		return (*nds)[i].NetworkID() >= nid
	})
	if i < len(*nds) && (*nds)[i].NetworkID() == nid {
		return i
	}
	return -1
}

func (nds *networkDigestSlice) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	for _, nd := range *nds {
		err := e2.Encode(nd.(*networkDigestFromBytes))
		if err != nil {
			return err
		}
	}
	return nil
}

func (nds *networkDigestSlice) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	ndSlice := make([]module.NetworkDigest, 0)
	for {
		var nd networkDigestFromBytes
		err := d2.Decode(&nd)
		if err == io.EOF {
			break
		}
		ndSlice = append(ndSlice, &nd)
	}
	*nds = ndSlice
	return nil
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
	messageBytes [][]byte,
	dbase db.Database,
	mod module.NetworkTypeModule,
) *messageList {
	hashesCat := hashesCat{
		Bytes: messageHashes,
	}
	messages := make([]*message, len(messageHashes)/hashLen)
	for i, bytes := range messageBytes {
		messages[i] = &message{
			dbase: dbase,
			mod:   mod,
			data:  bytes,
			hash:  hashesCat.Get(i),
		}
	}
	return &messageList{
		hashesCat: hashesCat,
		dbase:     dbase,
		mod:       mod,
		messages:  messages,
	}
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
	bk, err := l.dbase.GetBucket(l.mod.BytesByHashBucket())
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
	bk, err := l.dbase.GetBucket(l.mod.ListByMerkleRootBucket())
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
	bk, err := m.dbase.GetBucket(m.mod.BytesByHashBucket())
	if err != nil {
		return err
	}
	return bk.Set(m.Hash(), m.Bytes())
}

var ZeroDigest = &digest{
	digestCore: zeroDigestCore{},
}

type zeroDigestCore struct {
}

func (bd zeroDigestCore) Bytes() []byte {
	return nil
}

func (bd zeroDigestCore) Hash() []byte {
	return nil
}

func (bd zeroDigestCore) NetworkTypeDigests() []module.NetworkTypeDigest {
	return nil
}

func (bd zeroDigestCore) Flush(dbase db.Database) error {
	return nil
}
