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

package ntm

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

type proofContextCore interface {
	NetworkTypeModule() module.NetworkTypeModule
	Bytes() []byte
	NewProofPart(decisionHash []byte, wp module.WalletProvider) (module.BTPProofPart, error)
	NewProofPartFromBytes(ppBytes []byte) (module.BTPProofPart, error)
	// VerifyPart returns validator index and error
	VerifyPart(decisionHash []byte, pp module.BTPProofPart) (int, error)
	NewProof() module.BTPProof
	NewProofFromBytes(proofBytes []byte) (module.BTPProof, error)
	Verify(decisionHash []byte, p module.BTPProof) error
	DSA() string
}

type proofContext struct {
	core proofContextCore
	hash *[]byte
}

func (pc *proofContext) Hash() []byte {
	if pc.hash == nil {
		var hash []byte
		pcBytes := pc.core.Bytes()
		if pcBytes != nil {
			hash = pc.core.NetworkTypeModule().Hash(pcBytes)
		}
		pc.hash = &hash
	}
	return *pc.hash
}

func (pc *proofContext) Bytes() []byte {
	return pc.core.Bytes()
}

func (pc *proofContext) NewProofPart(decisionHash []byte, wp module.WalletProvider) (module.BTPProofPart, error) {
	return pc.core.NewProofPart(decisionHash, wp)
}

func (pc *proofContext) NewProofPartFromBytes(ppBytes []byte) (module.BTPProofPart, error) {
	return pc.core.NewProofPartFromBytes(ppBytes)
}

func (pc *proofContext) VerifyPart(decisionHash []byte, pp module.BTPProofPart) (int, error) {
	return pc.core.VerifyPart(decisionHash, pp)
}

func (pc *proofContext) NewProof() module.BTPProof {
	return pc.core.NewProof()
}

func (pc *proofContext) NewProofFromBytes(proofBytes []byte) (module.BTPProof, error) {
	return pc.core.NewProofFromBytes(proofBytes)
}

func (pc *proofContext) Verify(decisionHash []byte, p module.BTPProof) error {
	return pc.core.Verify(decisionHash, p)
}

func (pc *proofContext) DSA() string {
	return pc.core.DSA()
}

func (pc *proofContext) UID() string {
	return pc.core.NetworkTypeModule().UID()
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

func (pc *proofContext) NewDecision(srcUID []byte, dstNTID int64, height int64, round int32, ntsHash []byte) module.BytesHasher {
	return &networkTypeSectionDecision{
		SrcNetworkID:           srcUID,
		DstType:                dstNTID,
		Height:                 height,
		Round:                  round,
		NetworkTypeSectionHash: ntsHash,
		mod:                    pc.core.NetworkTypeModule(),
	}
}
