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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type NetworkType struct {
	uid                  string
	nextProofContextHash []byte
	nextProofContext     []byte
	openNetworkIDs       []int64
}

func (nt *NetworkType) UID() string {
	return nt.uid
}

func (nt *NetworkType) NextProofContextHash() []byte {
	return nt.nextProofContextHash
}

func (nt *NetworkType) NextProofContext() []byte {
	return nt.nextProofContext
}

func (nt *NetworkType) OpenNetworkIDs() []int64 {
	return nt.openNetworkIDs
}

func (nt *NetworkType) SetNextProofContextHash(hash []byte) {
	nt.nextProofContextHash = hash
}

func (nt *NetworkType) SetNextProofContext(bs []byte) {
	nt.nextProofContext = bs
}

func (nt *NetworkType) AddOpenNetworkID(nid int64) {
	nt.openNetworkIDs = append(nt.openNetworkIDs, nid)
}

func (nt *NetworkType) RemoveOpenNetworkID(nid int64) error {
	for i, v := range nt.OpenNetworkIDs() {
		if v == nid {
			copy(nt.openNetworkIDs[i:], nt.openNetworkIDs[i+1:])
			nt.openNetworkIDs[len(nt.openNetworkIDs)-1] = 0
			nt.openNetworkIDs = nt.openNetworkIDs[:len(nt.openNetworkIDs)-1]
			return nil
		}
	}
	return errors.Errorf("There is no open network id %d", nid)
}

func (nt *NetworkType) Bytes() []byte {
	return codec.MustMarshalToBytes(nt)
}

func (nt *NetworkType) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&nt.uid,
		&nt.nextProofContextHash,
		&nt.nextProofContext,
		&nt.openNetworkIDs,
	)
}

func (nt *NetworkType) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		nt.uid,
		nt.nextProofContextHash,
		nt.nextProofContext,
		nt.openNetworkIDs,
	)
}

func NewNetworkType(uid string, proofContext module.BTPProofContext) *NetworkType {
	nt := new(NetworkType)
	nt.uid = uid
	if proofContext != nil {
		nt.nextProofContext = proofContext.Bytes()
		nt.nextProofContextHash = proofContext.Hash()
	}
	return nt
}

func NewNetworkTypeFromBytes(b []byte) *NetworkType {
	nt := new(NetworkType)
	codec.MustUnmarshalFromBytes(b, nt)
	return nt
}

type Network struct {
	name                    string
	owner                   *common.Address
	networkTypeID           int64
	open                    bool
	nextMessageSN           int64
	nextProofContextChanged bool
	prevNetworkSectionHash  []byte
	lastNetworkSectionHash  []byte
}

func (nw *Network) Name() string {
	return nw.name
}

func (nw *Network) Owner() module.Address {
	return nw.owner
}

func (nw *Network) NetworkTypeID() int64 {
	return nw.networkTypeID
}

func (nw *Network) Open() bool {
	return nw.open
}

func (nw *Network) NextMessageSN() int64 {
	return nw.nextMessageSN
}

func (nw *Network) NextProofContextChanged() bool {
	return nw.nextProofContextChanged
}

func (nw *Network) PrevNetworkSectionHash() []byte {
	return nw.prevNetworkSectionHash
}

func (nw *Network) LastNetworkSectionHash() []byte {
	return nw.lastNetworkSectionHash
}

func (nw *Network) SetOpen(yn bool) {
	nw.open = yn
}

func (nw *Network) IncreaseNextMessageSN() {
	nw.nextMessageSN++
}

// TODO reset
func (nw *Network) SetNextProofContextChanged(yn bool) {
	nw.nextProofContextChanged = yn
}

func (nw *Network) SetPrevNetworkSectionHash(hash []byte) {
	nw.prevNetworkSectionHash = hash
}

func (nw *Network) SetLastNetworkSectionHash(hash []byte) {
	nw.lastNetworkSectionHash = hash
}

func (nw *Network) Bytes() []byte {
	return codec.MustMarshalToBytes(nw)
}

func (nw *Network) RLPDecodeSelf(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&nw.name,
		&nw.owner,
		&nw.networkTypeID,
		&nw.open,
		&nw.nextMessageSN,
		&nw.nextProofContextChanged,
		&nw.prevNetworkSectionHash,
		&nw.lastNetworkSectionHash,
	)
}

func (nw *Network) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		nw.name,
		nw.owner,
		nw.networkTypeID,
		nw.open,
		nw.nextMessageSN,
		nw.nextProofContextChanged,
		nw.prevNetworkSectionHash,
		nw.lastNetworkSectionHash,
	)
}

func NewNetwork(ntid int64, name string, owner module.Address, nextProofContextChanged bool) *Network {
	return &Network{
		networkTypeID:           ntid,
		name:                    name,
		owner:                   common.AddressToPtr(owner),
		open:                    true,
		nextProofContextChanged: nextProofContextChanged,
	}
}

func NewNetworkFromBytes(b []byte) (*Network, error) {
	nw := new(Network)
	_, err := codec.UnmarshalFromBytes(b, nw)
	if err != nil {
		return nil, err
	}
	return nw, nil
}
