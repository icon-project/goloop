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
	"bytes"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type ethProofPart struct {
	Index     int
	Signature common.Signature
}

func (pp *ethProofPart) Bytes() []byte {
	return codec.MustMarshalToBytes(pp)
}

func (pp *ethProofPart) recover(hash []byte) (ethAddress, error) {
	pubKey, err := pp.Signature.RecoverPublicKey(hash)
	if err != nil {
		return nil, err
	}
	return newEthAddressFromPubKey(pubKey.SerializeUncompressed()), nil
}

type ethProof struct {
	Signatures []common.Signature
	bytes      []byte
}

func (p *ethProof) Bytes() []byte {
	if p.bytes == nil {
		p.bytes = codec.MustMarshalToBytes(p)
	}
	return p.bytes
}

func (p *ethProof) Add(pp module.BTPProofPart) {
	epp := pp.(*ethProofPart)
	p.Signatures[epp.Index] = epp.Signature
}

type ethProofContext struct {
	Validators  []ethAddress
	bytes       []byte
	hash        []byte
	addrToIndex map[string]int
}

func newEthProofContext(
	pubKeys [][]byte,
) *ethProofContext {
	pp := &ethProofContext{
		Validators:  make([]ethAddress, 0, len(pubKeys)),
		addrToIndex: make(map[string]int, len(pubKeys)),
	}
	for i, pk := range pubKeys {
		addr := newEthAddressFromPubKey(pk)
		pp.Validators = append(pp.Validators, addr)
		pp.addrToIndex[string(addr)] = i
	}
	return pp
}

func (pc *ethProofContext) indexOf(address ethAddress) (int, bool) {
	if pc.addrToIndex == nil {
		pc.addrToIndex = make(map[string]int, len(pc.Validators))
		for i, addr := range pc.Validators {
			pc.addrToIndex[string(addr)] = i
		}
	}
	idx, ok := pc.addrToIndex[string(address)]
	return idx, ok
}

func newEthProofContextFromBytes(
	bytes []byte,
) (*ethProofContext, error) {
	pc := &ethProofContext{}
	_, err := codec.UnmarshalFromBytes(bytes, pc)
	if err != nil {
		return nil, err
	}
	return pc, nil
}

func (pc *ethProofContext) Hash() []byte {
	if pc.hash == nil {
		pc.hash = keccak256(pc.Bytes())
	}
	return pc.hash
}

func (pc *ethProofContext) Bytes() []byte {
	if pc.bytes == nil {
		pc.bytes = codec.MustMarshalToBytes(pc)
	}
	return pc.bytes
}

func (pc *ethProofContext) VerifyPart(dHash []byte, pp module.BTPProofPart) error {
	epp := pp.(*ethProofPart)
	if epp.Index < 0 || epp.Index >= len(pc.Validators) {
		return errors.Errorf("invalid proof part index=%d numValidators=%d", epp.Index, len(pc.Validators))
	}
	addr, err := epp.recover(dHash)
	if err != nil {
		return err
	}
	if !bytes.Equal(pc.Validators[epp.Index], addr) {
		return errors.Errorf("invalid proof part index=%d addr=%x", epp.Index, addr)
	}
	return nil
}

func (pc *ethProofContext) Verify(dHash []byte, p module.BTPProof) error {
	ep := p.(*ethProof)
	set := make(map[int]struct{}, len(ep.Signatures))
	valid := 0
	for i, sig := range ep.Signatures {
		if sig.Signature == nil {
			continue
		}
		epp := ethProofPart{
			Index:     i,
			Signature: sig,
		}
		err := pc.VerifyPart(dHash, &epp)
		if err != nil {
			return err
		}
		if _, ok := set[epp.Index]; ok {
			addr, _ := epp.recover(dHash)
			return errors.Errorf("duplicated proof parts validator index=%d addr=%x", epp.Index, addr)
		} else {
			set[epp.Index] = struct{}{}
		}
		valid++
	}
	if valid <= 2*len(pc.Validators)/3 {
		return errors.Errorf("not enough proof parts numValidator=%d numProofParts=%d", len(pc.Validators), len(ep.Signatures))
	}
	return nil
}

func (pc *ethProofContext) VerifyByProofBytes(dHash []byte, proofBytes []byte) error {
	var p ethProof
	_, err := codec.UnmarshalFromBytes(proofBytes, &p)
	if err != nil {
		return err
	}
	return pc.Verify(dHash, &p)
}

func (pc *ethProofContext) NewProofPart(
	dHash []byte,
	wp module.WalletProvider,
) (module.BTPProofPart, error) {
	w := wp.WalletFor(ethUID)
	if w == nil {
		w = wp.WalletFor(ethDSA)
	}
	sig, err := w.Sign(dHash)
	if err != nil {
		return nil, err
	}
	addr := newEthAddressFromPubKey(w.PublicKey())
	idx, ok := pc.indexOf(addr)
	if !ok {
		return nil, errors.Errorf("not validator addr=%x", addr)
	}
	pp := ethProofPart{
		Index: idx,
	}
	err = pp.Signature.UnmarshalBinary(sig)
	log.Must(err)
	return &pp, nil
}

func (pc *ethProofContext) DSA() string {
	return ethDSA
}

func (pc *ethProofContext) NewProof() module.BTPProof {
	return &ethProof{
		Signatures: make([]common.Signature, len(pc.Validators)),
	}
}
