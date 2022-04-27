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
	"github.com/icon-project/goloop/module"
)

const hashLen = 32

type digestFormat struct {
	NetworkTypeDigests []*networkTypeDigest
}

type digest struct {
	digestFormat
	dbase              db.Database
	bytes              []byte
	hash               []byte
	networkTypeDigests []module.NetworkTypeDigest
}

func (bd *digest) Bytes() []byte {
	if bd.bytes == nil {
		bd.bytes = codec.MustMarshalToBytes(bd)
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
		bd.networkTypeDigests = make([]module.NetworkTypeDigest, 0, len(bd.digestFormat.NetworkTypeDigests))
		for _, ntd := range bd.digestFormat.NetworkTypeDigests {
			bd.networkTypeDigests = append(bd.networkTypeDigests, ntd)
		}
	}
	return bd.networkTypeDigests
}

func (bd *digest) Flush() error {
	bk, err := bd.dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return err
	}
	return bk.Set(bd.Hash(), bd.Bytes())
}

func (bd *digest) FlushAll() error {
	//TODO implement me
	panic("implement me")
}

type networkTypeDigestFormat struct {
	NetworkTypeID          int64
	NetworkTypeSectionHash []byte
	NetworkDigests         []networkDigest
}

type networkTypeDigest struct {
	format         networkTypeDigestFormat
	networkDigests []module.NetworkDigest
	mod            ntm.Module
	dbase          db.Database
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

func (ntd *networkTypeDigest) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&ntd.format)
}

func (ntd *networkTypeDigest) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&ntd.format)
}

type format struct {
	NetworkID          int64
	NetworkSectionHash []byte
	MessagesRoot       []byte
}

type networkDigest struct {
	format
	messageList *messageList
	mod         ntm.Module
	dbase       db.Database
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

func (nd *networkDigest) MessageList() (module.BTPMessageList, error) {
	bk, err := nd.dbase.GetBucket(db.ListByMerkleRootFor(nd.mod.UID()))
	if err != nil {
		return nil, err
	}
	bs, err := bk.Get(nd.format.MessagesRoot)
	if err != nil {
		return nil, err
	}
	return newMessageList(bs, nd.mod, nd.dbase), nil
}

func (nd *networkDigest) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(&nd.format)
}

func (nd *networkDigest) RLPDecodeSelf(d codec.Decoder) error {
	return d.Decode(&nd.format)
}

type messageList struct {
	MessageHashes []byte
	messages      []*message
	messagesRoot  []byte
	mod           ntm.Module
	dbase         db.Database
}

func newMessageList(
	messageHashes []byte,
	mod ntm.Module,
	dbase db.Database,
) *messageList {
	l := &messageList{
		MessageHashes: messageHashes,
		messages:      make([]*message, len(messageHashes)/hashLen),
		mod:           mod,
		dbase:         dbase,
	}
	return l
}

func (l *messageList) Bytes() []byte {
	return l.MessageHashes
}

func (l *messageList) MessagesRoot() []byte {
	if l.messagesRoot == nil {
		l.messagesRoot = l.mod.MerkleRootHashCat(l.MessageHashes)
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
	msgHash := l.MessageHashes[idx*hashLen : (idx+1)*hashLen]
	bs, err := bk.Get(msgHash)
	if err != nil {
		return nil, err
	}
	m := &message{
		bytes: bs,
		hash:  msgHash,
		mod:   l.mod,
		dbase: l.dbase,
	}
	_, err = codec.UnmarshalFromBytes(bs, m)
	if err != nil {
		return nil, err
	}
	l.messages[idx] = m
	return m, nil
}

func (l *messageList) Flush() error {
	bk, err := l.dbase.GetBucket(db.ListByMerkleRootFor(l.mod.UID()))
	if err != nil {
		return err
	}
	return bk.Set(l.MessagesRoot(), l.Bytes())
}

func (l *messageList) FlushAll() error {
	bk, err := l.dbase.GetBucket(db.ListByMerkleRootFor(l.mod.UID()))
	if err != nil {
		return err
	}
	err = bk.Set(l.MessagesRoot(), l.Bytes())
	if err != nil {
		return err
	}
	for _, m := range l.messages {
		err = m.Flush()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *messageList) Add(msg *messageFormat) {
	m := &message{
		messageFormat: *msg,
		mod:           l.mod,
		dbase:         l.dbase,
	}
	l.MessageHashes = append(l.MessageHashes, m.Hash()...)
	l.messages = append(l.messages, m)
	l.messagesRoot = nil
}

func (l *messageList) Len() int {
	return len(l.messages)
}

type messageFormat struct {
	Data []byte
}

type message struct {
	messageFormat
	bytes []byte
	hash  []byte
	mod   ntm.Module
	dbase db.Database
}

func (m *message) Hash() []byte {
	if m.hash == nil {
		m.hash = m.mod.Hash(m.bytes)
	}
	return m.hash
}

func (m *message) Bytes() []byte {
	if m.bytes == nil {
		m.bytes = codec.MustMarshalToBytes(m)
	}
	return m.bytes
}

func (m *message) Data() []byte {
	return m.messageFormat.Data
}

func (m *message) Flush() error {
	bk, err := m.dbase.GetBucket(db.BytesByHashFor(m.mod.UID()))
	if err != nil {
		return err
	}
	return bk.Set(m.Hash(), m.Bytes())
}
