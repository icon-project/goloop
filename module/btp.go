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

// Proof

type BTPProofPart interface {
	Bytes() []byte
}

type BTPProof interface {
	Bytes() []byte
	Add(pp BTPProofPart) error
}

type BTPProofContext interface {
	Hash() []byte
	Bytes() []byte
	VerifyPart(d *NetworkTypeSectionDecision, pp BTPProofPart) error
	Verify(d *NetworkTypeSectionDecision, p BTPProof) error
	NewProofPart(d *NetworkTypeSectionDecision, w BaseWallet) (BTPProofPart, error)
	DSA() string
}

type NetworkTypeSectionDecision struct {
	SrcNetworkID           []byte
	DstType                int32
	Height                 int64
	Round                  int32
	NetworkTypeSectionHash []byte
}

type NetworkTypeSectionDecisionProof struct {
	NetworkTypeSectionHash []byte
	Proof                  BTPProof
}

// Digest

type BTPDigest interface {
	Hash() []byte
	NetworkTypeDigests() []NetworkTypeDigest
	Flush() error
}

type NetworkTypeDigest interface {
	NetworkTypeID() int32
	NetworkTypeSectionHash() []byte
	NetworkDigests() []NetworkDigest
}

type NetworkDigest interface {
	NetworkID() int32
	NetworkSectionHash() []byte
	MessageList() MessageList
}

type MessageList interface {
	MessagesRoot() []byte
	Get(idx int) Message
	Flush() error
}

type Message interface {
	Hash() []byte
	Bytes() []byte
	Flush() error
}

// Section

type BTPSection interface {
	Digest() BTPDigest
	NetworkTypeSections() []NetworkTypeSection
}

type NetworkTypeSection interface {
	NetworkTypeID() int32
	Hash() []byte
	NetworkSectionsRoot() []byte
	NextProofContext() BTPProofContext
	NetworkSections() []NetworkSection
}

type NetworkSection interface {
	Hash() []byte
	NetworkID() int32
	MessageRootNumber() int64
	MessageRootSN() int64
	UpdatedNextProofContextHash() bool
	PrevHash() []byte
	MessageCount() int32
	MessagesRoot() []byte
}
