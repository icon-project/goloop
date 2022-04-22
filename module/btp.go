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

import "github.com/icon-project/goloop/common/db"

// Proof

type BTPProofPart interface {
	Bytes() []byte
}

type BTPProof interface {
	Bytes() []byte
	Add(pp BTPProofPart)
}

type BTPProofContext interface {
	Hash() []byte
	Bytes() []byte
	VerifyPart(decisionHash []byte, pp BTPProofPart) error
	Verify(decisionHash []byte, p BTPProof) error
	VerifyByProofBytes(decisionHash []byte, proofBytes []byte) error
	NewProofPart(decisionHash []byte, w BaseWallet) (BTPProofPart, error)
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
	Flush() error
	FlushAll() error
}

type NetworkTypeDigest interface {
	NetworkTypeID() int64
	NetworkTypeSectionHash() []byte
	NetworkDigests() []NetworkDigest
}

type NetworkDigest interface {
	NetworkID() int64
	NetworkSectionHash() []byte
	MessagesRoot() []byte
	MessageList() (BTPMessageList, error)
}

type BTPMessageList interface {
	Bytes() []byte
	MessagesRoot() []byte
	Get(idx int) (BTPMessage, error)
	Flush() error
	FlushAll() error
}

type BTPMessage interface {
	Hash() []byte
	Bytes() []byte
	Flush() error
}

// Section

type BTPSection interface {
	Digest(dbase db.Database) BTPDigest
	NetworkTypeSections() []NetworkTypeSection
}

type NetworkTypeSection interface {
	NetworkTypeID() int64
	Hash() []byte
	NetworkSectionsRoot() []byte
	NextProofContext() BTPProofContext
	NetworkSections() []NetworkSection
	NewDecision(height int64, round int32) BytesHasher
}

type BytesHasher interface {
	Bytes() []byte
	Hash() []byte
}

type NetworkSection interface {
	Hash() []byte
	NetworkID() int64
	MessageRootNumber() int64
	MessageRootSN() int64
	UpdatedNextProofContextHash() bool
	PrevHash() []byte
	MessageCount() int64
	MessagesRoot() []byte
}
