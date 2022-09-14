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
	ValidatorCount() int
	ProofPartAt(i int) BTPProofPart
}

type WalletProvider interface {
	// WalletFor returns public key for dsa.
	WalletFor(dsa string) BaseWallet
}

type BTPProofContext interface {
	Hash() []byte
	Bytes() []byte
	NewProofPart(decisionHash []byte, wp WalletProvider) (BTPProofPart, error)
	NewProofPartFromBytes(ppBytes []byte) (BTPProofPart, error)
	VerifyPart(decisionHash []byte, pp BTPProofPart) (int, error)
	NewProof() BTPProof
	NewProofFromBytes(proofBytes []byte) (BTPProof, error)
	Verify(decisionHash []byte, p BTPProof) error
	DSA() string
	UID() string
	NewDecision(
		srcNetworkUID []byte,
		ntid int64,
		height int64,
		round int32,
		ntsHash []byte,
	) BytesHasher
}

type NTSDProofList interface {
	NTSDProofCount() int
	NTSDProofAt(i int) []byte
}

type BTPProofContextMapUpdateSource interface {
	BTPSection() (BTPSection, error)
	NextProofContextMap() (BTPProofContextMap, error)
}

type BTPProofContextMap interface {
	ProofContextFor(ntid int64) (BTPProofContext, error)
	Update(src BTPProofContextMapUpdateSource) (BTPProofContextMap, error)
	Verify(
		srcUID []byte, height int64, round int32, bd BTPDigest,
		ntsdProves NTSDProofList,
	) error
}

// Digest

type BTPDigest interface {
	Bytes() []byte
	Hash() []byte
	NetworkTypeDigests() []NetworkTypeDigest
	NetworkTypeDigestFor(ntid int64) NetworkTypeDigest

	// NetworkTypeIDFromNID returns ntid for nid in this digest. Returns
	// wrapped ErrNotFound if nid is not found in this digest.
	NetworkTypeIDFromNID(nid int64) (int64, error)

	// Flush writes this BTPDigest and its connected objects.
	// If a BTPDigest is created by a BTPSection and the BTPSection is created
	// by btp.SectionBuilder, the BTPDigest has all the BTPMessageList's and
	// the BTPMessage's in the section as its connected objects. Thus, they are
	// all written when Flush is called. In other cases, a BTPDigest has no
	// connected objects. Thus, only the BTPDigest itself is written when Flush
	// is called.
	Flush(dbase db.Database) error
	NetworkSectionFilter() BitSetFilter
	NTSHashEntryList

	// NTSVoteCount returns NTS vote count for this digest.
	// pcm is the nextPCM in block(h-1) where the digest is in block(h).
	NTSVoteCount(pcm BTPProofContextMap) (int, error)
}

type NetworkTypeDigest interface {
	NetworkTypeID() int64
	NetworkTypeSectionHash() []byte
	NetworkDigests() []NetworkDigest
	NetworkDigestFor(nid int64) NetworkDigest
	NetworkSectionsRootWithMod(mod NetworkTypeModule) []byte
	NetworkSectionToRootWithMod(mod NetworkTypeModule, nid int64) ([]MerkleNode, error)
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
	NetworkTypeSectionFor(ntid int64) (NetworkTypeSection, error)
}

type NetworkTypeSection interface {
	NetworkTypeID() int64
	Hash() []byte
	NetworkSectionsRoot() []byte
	NetworkSectionToRoot(nid int64) ([]MerkleNode, error)
	NextProofContext() BTPProofContext
	NetworkSectionFor(nid int64) (NetworkSection, error)
	NewDecision(srcNetworkUID []byte, height int64, round int32) BytesHasher
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

type BytesSlice [][]byte

func (b *BytesSlice) Len() int {
	return len(*b)
}

func (b *BytesSlice) Get(i int) []byte {
	return (*b)[i]
}

type BytesList interface {
	Len() int
	Get(i int) []byte
}

type Dir int

const (
	DirLeft = Dir(iota)
	DirRight
)

type MerkleNode struct {
	Dir   Dir
	Value []byte
}

type NetworkTypeModule interface {
	UID() string
	Hash(data []byte) []byte
	AppendHash(out []byte, data []byte) []byte
	DSA() string
	NewProofContextFromBytes(bs []byte) (BTPProofContext, error)

	// NewProofContext returns a new proof context. The parameter keys is
	// a slice of networkType specific keys (usually a slice of addresses).
	NewProofContext(keys [][]byte) (BTPProofContext, error)
	MerkleRoot(bytesList BytesList) []byte
	MerkleProof(bytesList BytesList, idx int) []MerkleNode
	AddressFromPubKey(pubKey []byte) ([]byte, error)
	BytesByHashBucket() db.BucketID
	ListByMerkleRootBucket() db.BucketID
	NewProofFromBytes(bs []byte) (BTPProof, error)
	NetworkTypeKeyFromDSAKey(key []byte) ([]byte, error)
	DSAModule() DSAModule
}

type BTPBlockHeader interface {
	MainHeight() int64
	Round() int32
	NextProofContextHash() []byte
	NetworkSectionToRoot() []MerkleNode
	NetworkID() int64
	UpdateNumber() int64
	FirstMessageSN() int64
	NextProofContextChanged() bool
	PrevNetworkSectionHash() []byte
	MessageCount() int64
	MessagesRoot() []byte
	NextProofContext() []byte
	HeaderBytes() []byte
}

type BTPNetworkType interface {
	UID() string
	NextProofContextHash() []byte
	NextProofContext() []byte
	OpenNetworkIDs() []int64
	ToJSON() map[string]interface{}
}

type BTPNetwork interface {
	StartHeight() int64
	Name() string
	Owner() Address
	NetworkTypeID() int64
	Open() bool
	NextMessageSN() int64
	NextProofContextChanged() bool
	PrevNetworkSectionHash() []byte
	LastNetworkSectionHash() []byte
	ToJSON() map[string]interface{}
}

type DSAModule interface {
	Name() string

	// Verify verifies format of public key.
	Verify(pubKey []byte) error
}
