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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/btp/ntm"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

func rlpListOf(s ...interface{}) []byte {
	var bs []byte
	e := codec.NewEncoderBytes(&bs)
	_ = e.EncodeListOf(s...)
	return bs
}

func hashOfRLPList(mod module.NetworkTypeModule, s ...interface{}) []byte {
	return mod.Hash(rlpListOf(s...))
}

type testStateView struct {
	networks     map[int64]*Network
	networkTypes map[int64]*NetworkType
}

func (v *testStateView) GetNetwork(nid int64) (*Network, error) {
	if nw, ok := v.networks[nid]; ok {
		return nw, nil
	}
	return nil, errors.ErrNotFound
}

func (v *testStateView) GetNetworkType(ntid int64) (*NetworkType, error) {
	if nt, ok := v.networkTypes[ntid]; ok {
		return nt, nil
	}
	return nil, errors.ErrNotFound
}

func TestSectionBuilder_Build_Empty(t *testing.T) {
	assert := assert.New(t)
	view := &testStateView{}
	builder := NewSectionBuilder(view)
	bs, err := builder.Build()
	assert.NoError(err)
	bd := bs.Digest()
	assert.EqualValues([]byte(nil), bd.Bytes())
}

func TestSectionBuilder_Build_Basic(t *testing.T) {
	assert := assert.New(t)
	mod := ntm.ForUID("eth")
	pc, err := mod.NewProofContext(nil)
	assert.NoError(err)
	view := &testStateView{
		networks: map[int64]*Network{
			2: {
				NetworkTypeID:           1,
				Open:                    true,
				NextMessageSN:           2,
				NextProofContextChanged: false,
				PrevNetworkSectionHash:  nil,
				LastNetworkSectionHash:  nil,
			},
		},
		networkTypes: map[int64]*NetworkType{
			1: {
				UID:                  "eth",
				NextProofContextHash: pc.Hash(),
				NextProofContext:     pc.Bytes(),
				OpenNetworkIDs:       []int64{1, 2},
			},
		},
	}
	builder := NewSectionBuilder(view)
	builder.EnsureSection(2)
	bs, err := builder.Build()
	assert.NoError(err)

	ntsSlice := bs.NetworkTypeSections()
	assert.EqualValues(1, len(ntsSlice))
	nts := ntsSlice[0]
	assert.EqualValues(1, nts.NetworkTypeID())
	nd := bs.Digest().NetworkTypeDigestFor(1).NetworkDigestFor(2)
	assert.EqualValues(2, nd.NetworkID())
	ns, _ := nts.NetworkSectionFor(2)
	assert.EqualValues(2, ns.NetworkID())

	nsHash := hashOfRLPList(mod, 2, 2<<1, nil, 0, nil)
	assert.EqualValues(nsHash, ns.Hash())

	ntsHash := hashOfRLPList(mod, pc.Hash(), nsHash)
	assert.EqualValues(ntsHash, nts.Hash())
}

type testBuilderSetup struct {
	mod     module.NetworkTypeModule
	pc      module.BTPProofContext
	view    *testStateView
	builder SectionBuilder
	bs      module.BTPSection
}

func newComplexTestBuilderSetup(t *testing.T) *testBuilderSetup {
	assert := assert.New(t)
	mod := ntm.ForUID("eth")
	pc, err := mod.NewProofContext(nil)
	assert.NoError(err)
	view := &testStateView{
		networks: map[int64]*Network{
			1: {
				NetworkTypeID:           1,
				Open:                    true,
				NextMessageSN:           1,
				NextProofContextChanged: false,
				PrevNetworkSectionHash:  nil,
				LastNetworkSectionHash:  nil,
			},
			2: {
				NetworkTypeID:           1,
				Open:                    true,
				NextMessageSN:           2,
				NextProofContextChanged: false,
				PrevNetworkSectionHash:  nil,
				LastNetworkSectionHash:  nil,
			},
			3: {
				NetworkTypeID:           2,
				Open:                    true,
				NextMessageSN:           3,
				NextProofContextChanged: false,
				PrevNetworkSectionHash:  nil,
				LastNetworkSectionHash:  nil,
			},
			4: {
				NetworkTypeID:           2,
				Open:                    true,
				NextMessageSN:           3,
				NextProofContextChanged: false,
				PrevNetworkSectionHash:  nil,
				LastNetworkSectionHash:  nil,
			},
		},
		networkTypes: map[int64]*NetworkType{
			1: {
				UID:                  "eth",
				NextProofContextHash: pc.Hash(),
				NextProofContext:     pc.Bytes(),
				OpenNetworkIDs:       []int64{1, 2},
			},
			2: {
				UID:                  "eth",
				NextProofContextHash: pc.Hash(),
				NextProofContext:     pc.Bytes(),
				OpenNetworkIDs:       []int64{3, 4},
			},
		},
	}
	builder := NewSectionBuilder(view)
	builder.EnsureSection(2)
	builder.SendMessage(1, []byte("a"))
	builder.EnsureSection(4)
	builder.SendMessage(3, []byte("b"))
	builder.SendMessage(3, []byte("c"))
	builder.SendMessage(3, []byte("d"))
	bs, err := builder.Build()
	assert.NoError(err)
	return &testBuilderSetup{
		mod:     mod,
		pc:      pc,
		view:    view,
		builder: builder,
		bs:      bs,
	}
}

func (s *testBuilderSetup) updateView() {
	d := s.bs.Digest()
	for _, ntd := range d.NetworkTypeDigests() {
		for _, nd := range ntd.NetworkDigests() {
			if nw, ok := s.view.networks[nd.NetworkID()]; ok {
				nw.PrevNetworkSectionHash = nw.LastNetworkSectionHash
				nw.LastNetworkSectionHash = nd.NetworkSectionHash()
			}
		}
	}
}

func TestSectionBuilder_Build_Complex(t *testing.T) {
	assert := assert.New(t)
	s := newComplexTestBuilderSetup(t)

	ntsSlice := s.bs.NetworkTypeSections()
	assert.EqualValues(2, len(ntsSlice))
	assert.EqualValues(1, ntsSlice[0].NetworkTypeID())
	assert.EqualValues(2, ntsSlice[1].NetworkTypeID())

	assert.EqualValues(1, s.bs.Digest().NetworkTypeDigestFor(1).NetworkDigestFor(1).NetworkID())
	assert.EqualValues(2, s.bs.Digest().NetworkTypeDigestFor(1).NetworkDigestFor(2).NetworkID())

	ns0, _ := ntsSlice[0].NetworkSectionFor(1)
	assert.EqualValues(1, ns0.NetworkID())
	ns1, _ := ntsSlice[0].NetworkSectionFor(2)
	assert.EqualValues(2, ns1.NetworkID())

	ns0Hash := hashOfRLPList(s.mod, 1, 0<<1, nil, 1, s.mod.Hash([]byte("a")))
	assert.EqualValues(ns0Hash, ns0.Hash())
	ns1Hash := hashOfRLPList(s.mod, 2, 2<<1, nil, 0, nil)
	assert.EqualValues(ns1Hash, ns1.Hash())

	nts0Hash := hashOfRLPList(
		s.mod,
		s.pc.Hash(),
		hashOfRLPList(
			s.mod,
			ns0Hash,
			ns1Hash,
		),
	)
	assert.EqualValues(nts0Hash, ntsSlice[0].Hash())

	ns2, _ := ntsSlice[1].NetworkSectionFor(3)
	ns3, _ := ntsSlice[1].NetworkSectionFor(4)
	ns2Hash := hashOfRLPList(s.mod, 3, 0, nil, 3, s.mod.MerkleRoot(
		&module.BytesSlice{
			s.mod.Hash([]byte("b")),
			s.mod.Hash([]byte("c")),
			s.mod.Hash([]byte("d")),
		},
	))
	assert.EqualValues(ns2Hash, ns2.Hash())
	ns3Hash := hashOfRLPList(s.mod, 4, 3<<1, nil, 0, nil)
	assert.EqualValues(ns3Hash, ns3.Hash())

	nts1Hash := hashOfRLPList(
		s.mod,
		s.pc.Hash(),
		hashOfRLPList(
			s.mod,
			ns2Hash,
			ns3Hash,
		),
	)
	assert.EqualValues(nts1Hash, ntsSlice[1].Hash())
}
