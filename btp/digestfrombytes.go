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

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/merkle"
	"github.com/icon-project/goloop/module"
)

type networkTypeDigestSlice []module.NetworkTypeDigest

func (ntds *networkTypeDigestSlice) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	for _, ntd := range *ntds {
		err = e2.Encode(ntd.(*networkTypeDigest).networkTypeDigestCore.(*networkTypeDigestCoreFromBytes))
		if err != nil {
			return err
		}
	}
	return nil
}

func (ntds *networkTypeDigestSlice) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}
	var res networkTypeDigestSlice
	for {
		var ntd networkTypeDigestCoreFromBytes
		err := d2.Decode(&ntd)
		if err == io.EOF {
			break
		}
		res = append(res, &networkTypeDigest{
			networkTypeDigestCore: &ntd,
		})
	}
	*ntds = res
	return nil
}

type digestFormat struct {
	NetworkTypeDigests networkTypeDigestSlice
}

type digestCoreFromBytes struct {
	bytes  []byte
	hash   []byte
	format digestFormat
}

func NewDigestFromHashAndBytes(
	hash []byte,
	bytes []byte,
) (module.BTPDigest, error) {
	core := &digestCoreFromBytes{
		bytes: bytes,
		hash:  hash,
	}
	if bytes != nil {
		_, err := codec.UnmarshalFromBytes(bytes, &core.format)
		if err != nil {
			return nil, err
		}
	}
	return &digest{
		core: core,
	}, nil
}

func NewDigestFromBytes(bytes []byte) (module.BTPDigest, error) {
	var hash []byte
	if bytes != nil {
		hash = crypto.SHA3Sum256(bytes)
	}
	return NewDigestFromHashAndBytes(hash, bytes)
}

func (bd *digestCoreFromBytes) Bytes() []byte {
	return bd.bytes
}

func (bd *digestCoreFromBytes) Hash() []byte {
	return bd.hash
}

func (bd *digestCoreFromBytes) NetworkTypeDigests() []module.NetworkTypeDigest {
	return bd.format.NetworkTypeDigests
}

func (bd *digestCoreFromBytes) Flush(dbase db.Database) error {
	if bd.bytes == nil {
		return nil
	}
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

type networkTypeDigestFormat struct {
	NetworkTypeID          int64
	UID                    string
	NetworkTypeSectionHash []byte
	NetworkDigests         networkDigestSlice
}

type networkTypeDigestCoreFromBytes struct {
	format              networkTypeDigestFormat
	networkSectionsRoot []byte
}

func (ntd *networkTypeDigestCoreFromBytes) NetworkTypeID() int64 {
	return ntd.format.NetworkTypeID
}

func (ntd *networkTypeDigestCoreFromBytes) UID() string {
	return ntd.format.UID
}

func (ntd *networkTypeDigestCoreFromBytes) NetworkTypeSectionHash() []byte {
	return ntd.format.NetworkTypeSectionHash
}

func (ntd *networkTypeDigestCoreFromBytes) NetworkDigests() []module.NetworkDigest {
	return ntd.format.NetworkDigests
}

func (ntd *networkTypeDigestCoreFromBytes) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&ntd.format)
}

func (ntd *networkTypeDigestCoreFromBytes) RLPDecodeSelf(d codec.Decoder) error {
	err := d.Decode(&ntd.format)
	ntd.networkSectionsRoot = nil
	return err
}

type networkDigestFormat struct {
	NetworkID          int64
	NetworkSectionHash []byte
	MessagesRoot       []byte
}

type networkDigestFromBytes struct {
	format        networkDigestFormat
	messageHashes []byte
}

func (nd *networkDigestFromBytes) NetworkID() int64 {
	return nd.format.NetworkID
}

func (nd *networkDigestFromBytes) NetworkSectionHash() []byte {
	return nd.format.NetworkSectionHash
}

func (nd *networkDigestFromBytes) MessagesRoot() []byte {
	return nd.format.MessagesRoot
}

func (nd *networkDigestFromBytes) MessageList(
	dbase db.Database,
	mod module.NetworkTypeModule,
) (module.BTPMessageList, error) {
	if nd.messageHashes == nil {
		bk, err := dbase.GetBucket(mod.ListByMerkleRootBucket())
		if err != nil {
			return nil, err
		}
		bs, err := bk.Get(nd.format.MessagesRoot)
		if err != nil {
			return nil, err
		}
		nd.messageHashes = bs
	}
	return newMessageList(nd.messageHashes, nil, dbase, mod), nil
}

func (nd *networkDigestFromBytes) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&nd.format)
}

func (nd *networkDigestFromBytes) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&nd.format)
}

type dummyHandler struct{}

func (h dummyHandler) OnData(value []byte, builder merkle.Builder) error {
	return nil
}

type messageDataHandler struct {
	mod module.NetworkTypeModule
}

func (h *messageDataHandler) OnData(value []byte, builder merkle.Builder) error {
	hc := &hashesCat{
		Bytes: value,
	}
	bkid := h.mod.BytesByHashBucket()
	for i := 0; i < hc.Len(); i++ {
		builder.RequestData(bkid, hc.Get(i), dummyHandler{})
	}
	return nil
}

type digestDataHandler struct {
	core   *digestCoreFromBytes
	digest *digest
}

func (h *digestDataHandler) OnData(value []byte, builder merkle.Builder) error {
	h.core.bytes = value
	_, err := codec.UnmarshalFromBytes(value, &h.core.format)
	if err != nil {
		return err
	}
	ntds := h.core.NetworkTypeDigests()
	for _, ntd := range ntds {
		nds := ntd.NetworkDigests()
		for _, nd := range nds {
			root := nd.MessagesRoot()
			mod := ntm.ForUID(ntd.UID())
			bk := mod.ListByMerkleRootBucket()
			builder.RequestData(bk, root, &messageDataHandler{
				mod: mod,
			})
		}
	}
	return nil
}

func NewDigestWithBuilder(builder merkle.Builder, hash []byte) (module.BTPDigest, error) {
	if hash == nil {
		return ZeroDigest, nil
	}
	core := &digestCoreFromBytes{
		hash: hash,
	}
	ret := &digest{
		core: core,
	}
	builder.RequestData(db.BytesByHash, hash, &digestDataHandler{
		core:   core,
		digest: ret,
	})
	return ret, nil
}
