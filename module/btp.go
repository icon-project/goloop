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

package module

import (
	"github.com/icon-project/goloop/common/db"
)

// Proof

type BTPProofPart interface {
	Bytes() []byte
}

type BTPProof interface {
	Bytes() []byte
	Add(pp BTPProofPart)
}

type WalletProvider interface {
	// WalletFor returns key for keyType. keyType can be network type uid or
	// DSA. For network type uid, network type specific key (usually address) is
	// returned.  For DSA, public key for the DSA is returned.
	WalletFor(keyType string) BaseWallet
}

type BTPProofContext interface {
	Hash() []byte
	Bytes() []byte
	VerifyPart(decisionHash []byte, pp BTPProofPart) error
	Verify(decisionHash []byte, p BTPProof) error
	VerifyByProofBytes(decisionHash []byte, proofBytes []byte) error
	NewProofPart(decisionHash []byte, wp WalletProvider) (BTPProofPart, error)
	DSA() string
	NewProof() BTPProof
}

type NetworkTypeSectionDecisionProof struct {
	NetworkTypeSectionHash []byte
	Proof                  BTPProof
}

// Digest

type BTPDigest interface {
	Bytes() []byte
	Hash() []byte
	NetworkTypeDigests() []NetworkTypeDigest
	NetworkTypeDigestFor(ntid int64) NetworkTypeDigest

	// Flush writes this BTPDigest and its connected objects.
	// If a BTPDigest is created by a BTPSection and the BTPSection is created
	// by btp.SectionBuilder, the BTPDigest has all the BTPMessageList's and
	// the BTPMessage's in the section as its connected objects. Thus, they are
	// all written when Flush is called. In other cases, a BTPDigest has no
	// connected objects. Thus, only the BTPDigest itself is written when Flush
	// is called.
	Flush(dbase db.Database) error
	NetworkSectionFilter() BitSetFilter
}

type NetworkTypeDigest interface {
	NetworkTypeID() int64
	NetworkTypeSectionHash() []byte
	NetworkDigests() []NetworkDigest
	NetworkDigestFor(nid int64) NetworkDigest
	NetworkSectionsRootWithMod(mod NetworkTypeModule) []byte
}

type NetworkDigest interface {
	NetworkID() int64
	NetworkSectionHash() []byte
	MessagesRoot() []byte
	MessageList(dbase db.Database, mod NetworkTypeModule) (BTPMessageList, error)
}

type BTPMessageList interface {
	Bytes() []byte
	MessagesRoot() []byte
	Len() int64
	Get(idx int) (BTPMessage, error)
}

type BTPMessage interface {
	Hash() []byte
	Bytes() []byte
}

// Section

type BTPSection interface {
	Digest() BTPDigest
	NetworkTypeSections() []NetworkTypeSection
	NetworkTypeSectionFor(ntid int64) NetworkTypeSection
}

type NetworkTypeSection interface {
	NetworkTypeID() int64
	Hash() []byte
	NetworkSectionsRoot() []byte
	NextProofContext() BTPProofContext
	NetworkSectionFor(nid int64) NetworkSection
	NewDecision(height int64, round int32) BytesHasher
}

type BytesHasher interface {
	Bytes() []byte
	Hash() []byte
}

type NetworkSection interface {
	Hash() []byte
	NetworkID() int64
	// UpdateNumber returns FirstMessageSN() << 1 | NextProofContextChanged()
	UpdateNumber() int64
	FirstMessageSN() int64
	NextProofContextChanged() bool
	PrevHash() []byte
	MessageCount() int64
	MessagesRoot() []byte
}

type BytesList interface {
	Len() int
	Get(i int) []byte
}

// NetworkTypeModule represents a network type module.
type NetworkTypeModule interface {
	UID() string
	Hash(data []byte) []byte
	DSA() string
	NewProofContextFromBytes(bs []byte) (BTPProofContext, error)
	NewProofContext(pubKeys [][]byte) BTPProofContext
	MerkleRoot(bytesList BytesList) []byte
}
