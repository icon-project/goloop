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
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

type walletProvider struct {
	wallets map[string]module.BaseWallet
}

func (w walletProvider) WalletFor(keyType string) module.BaseWallet {
	return w.wallets[keyType]
}

func newPCs(assert *assert.Assertions, count int, uids ...string) []module.BTPProofContext {
	pcs := make([]module.BTPProofContext, len(uids))
	addrs := make([][][]byte, len(uids))
	for i := 0; i < count; i++ {
		w := wallet.New()
		for i, uid := range uids {
			addr, err := ntm.ForUID(uid).AddressFromPubKey(w.PublicKey())
			assert.NoError(err)
			addrs[i] = append(addrs[i], addr)
		}
	}
	for i, uid := range uids {
		var err error
		pcs[i], err = ntm.ForUID(uid).NewProofContext(addrs[i])
		assert.NoError(err)
	}
	return pcs
}

type sectionPCMUpdateSource struct {
	btpSection module.BTPSection
}

func (s sectionPCMUpdateSource) BTPSection() (module.BTPSection, error) {
	return s.btpSection, nil
}

func (s sectionPCMUpdateSource) NextProofContextMap() (module.BTPProofContextMap, error) {
	return nil, nil
}

func TestProofContextMap_Basic(t *testing.T) {
	const count = 4
	assert := assert.New(t)

	pcs := newPCs(assert, count, "eth", "icon")
	view := &testStateView{
		networkTypeIDs: []int64{1, 2},
		networks: map[int64]*network{
			1: {
				networkTypeID:           1,
				open:                    true,
				nextMessageSN:           1,
				nextProofContextChanged: false,
				prevNetworkSectionHash:  nil,
				lastNetworkSectionHash:  nil,
			},
			2: {
				networkTypeID:           2,
				open:                    true,
				nextMessageSN:           1,
				nextProofContextChanged: false,
				prevNetworkSectionHash:  nil,
				lastNetworkSectionHash:  nil,
			},
		},
		networkTypes: map[int64]*networkType{
			1: {
				uid:                  "eth",
				nextProofContextHash: pcs[0].Hash(),
				nextProofContext:     pcs[0].Bytes(),
				openNetworkIDs:       []int64{1},
			},
			2: {
				uid:                  "icon",
				nextProofContextHash: pcs[1].Hash(),
				nextProofContext:     pcs[1].Bytes(),
				openNetworkIDs:       []int64{2},
			},
		},
	}
	pcm, err := NewProofContextsMap(view)
	assert.NoError(err)

	_, err = pcm.ProofContextFor(0)
	assert.Error(err)

	pc0, err := pcm.ProofContextFor(1)
	assert.NoError(err)
	assert.EqualValues(pcs[0].Bytes(), pc0.Bytes())

	pc1, err := pcm.ProofContextFor(2)
	assert.NoError(err)
	assert.EqualValues(pcs[1].Bytes(), pc1.Bytes())

	pcs = newPCs(assert, count, "eth", "icon")

	view = &testStateView{
		networks: map[int64]*network{
			1: {
				networkTypeID:           1,
				open:                    true,
				nextMessageSN:           1,
				nextProofContextChanged: true,
				prevNetworkSectionHash:  nil,
				lastNetworkSectionHash:  nil,
			},
			2: {
				networkTypeID:           2,
				open:                    true,
				nextMessageSN:           1,
				nextProofContextChanged: true,
				prevNetworkSectionHash:  nil,
				lastNetworkSectionHash:  nil,
			},
		},
		networkTypes: map[int64]*networkType{
			1: {
				uid:                  "eth",
				nextProofContextHash: pcs[0].Hash(),
				nextProofContext:     pcs[0].Bytes(),
				openNetworkIDs:       []int64{1},
			},
			2: {
				uid:                  "icon",
				nextProofContextHash: pcs[1].Hash(),
				nextProofContext:     pcs[1].Bytes(),
				openNetworkIDs:       []int64{2},
			},
		},
	}
	builder := NewSectionBuilder(view)
	builder.EnsureSection(1)
	builder.EnsureSection(2)
	bs, err := builder.Build()
	assert.NoError(err)
	pcm2, err := pcm.Update(sectionPCMUpdateSource{bs})
	assert.NoError(err)

	_, err = pcm2.ProofContextFor(0)
	assert.Error(err)

	pc0, err = pcm2.ProofContextFor(1)
	assert.NoError(err)
	assert.EqualValues(pcs[0].Bytes(), pc0.Bytes())

	pc1, err = pcm2.ProofContextFor(2)
	assert.NoError(err)
	assert.EqualValues(pcs[1].Bytes(), pc1.Bytes())
}
