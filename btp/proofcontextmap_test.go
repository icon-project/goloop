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

func newKeys(t *testing.T, count int, uids ...string) ([]module.BTPProofContext, []module.WalletProvider, [][][]byte, [][][]byte) {
	assert := assert.New(t)
	pcs := make([]module.BTPProofContext, len(uids))
	wps := make([]module.WalletProvider, 0, count)
	pks := make([][][]byte, len(uids))
	addrs := make([][][]byte, len(uids))
	for i := 0; i < count; i++ {
		w := wallet.New()
		wp := walletProvider{
			make(map[string]module.BaseWallet),
		}
		wps = append(wps, wp)
		wp.wallets["ecdsa/secp256k1"] = w
		for j, uid := range uids {
			pks[j] = append(pks[j], w.PublicKey())
			addr, err := ntm.ForUID(uid).AddressFromPubKey(w.PublicKey())
			assert.NoError(err)
			addrs[j] = append(addrs[j], addr)
		}
	}
	for i, uid := range uids {
		var err error
		pcs[i], err = ntm.ForUID(uid).NewProofContext(pks[i])
		assert.NoError(err)
	}
	return pcs, wps, pks, addrs
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

type ntsdProofList [][]byte

func (l ntsdProofList) NTSDProofCount() int {
	return len(l)
}

func (l ntsdProofList) NTSDProofAt(i int) []byte {
	return l[i]
}

type pcmTest struct {
	*assert.Assertions
	*testing.T
	PCs   []module.BTPProofContext
	WPs   []module.WalletProvider
	PKs   [][][]byte
	Addrs [][][]byte
	View  *testStateView
	PCM   module.BTPProofContextMap
	UIDs  []string
}

func newPCMTest(t *testing.T) *pcmTest {
	uids := []string{"eth", "icon"}
	const count = 4
	assert := assert.New(t)

	pcs, wps, pks, addrs := newKeys(t, count, uids...)
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

	return &pcmTest{
		Assertions: assert,
		T:          t,
		PCs:        pcs,
		WPs:        wps,
		PKs:        pks,
		Addrs:      addrs,
		View:       view,
		PCM:        pcm,
		UIDs:       uids,
	}
}

func TestProofContextMap_ProofContextForError(t_ *testing.T) {
	t := newPCMTest(t_)
	_, err := t.PCM.ProofContextFor(0)
	t.Error(err)
}

type pcmVerifyTest struct {
	*pcmTest
	Height        int64
	Round         int64
	BS            module.BTPSection
	BD            module.BTPDigest
	NTSDProofList ntsdProofList
	SrcUID        []byte
}

func newPCMVerifyTest(t_ *testing.T) *pcmVerifyTest {
	const count = 4
	assert := assert.New(t_)

	t := newPCMTest(t_)

	pc0, err := t.PCM.ProofContextFor(1)
	assert.NoError(err)
	assert.EqualValues(t.PCs[0].Bytes(), pc0.Bytes())

	pc1, err := t.PCM.ProofContextFor(2)
	assert.NoError(err)
	assert.EqualValues(t.PCs[1].Bytes(), pc1.Bytes())

	pcs, _, _, _ := newKeys(t_, count, "eth", "icon")

	view := &testStateView{
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

	bd := bs.Digest()
	srcUID := module.SourceNetworkUID(0)
	var ntsdPL ntsdProofList
	for i, pc := range []module.BTPProofContext{pc0, pc1} {
		pf := pc.NewProof()
		ntid := int64(i + 1)
		nts, err := bs.NetworkTypeSectionFor(ntid)
		assert.NoError(err)
		dcs := pc.NewDecision(srcUID, ntid, 3, 0, nts.Hash())
		for j := 0; j < count; j++ {
			pp, err := pc.NewProofPart(dcs.Hash(), t.WPs[j])
			assert.NoError(err)
			pf.Add(pp)
		}
		ntsdPL = append(ntsdPL, pf.Bytes())
	}
	return &pcmVerifyTest{
		pcmTest:       t,
		Height:        3,
		Round:         0,
		BS:            bs,
		BD:            bd,
		NTSDProofList: ntsdPL,
		SrcUID:        srcUID,
	}
}

func TestProofContextMap_VerifyOK(t_ *testing.T) {
	t := newPCMVerifyTest(t_)
	err := t.PCM.Verify(t.SrcUID, 3, 0, t.BD, t.NTSDProofList)
	t.NoError(err)
}

func TestProofContextMap_VerifyShort(t_ *testing.T) {
	t := newPCMVerifyTest(t_)
	pl := t.NTSDProofList[:1]
	err := t.PCM.Verify(t.SrcUID, 3, 0, t.BD, pl)
	t.Error(err)
}

func TestProofContextMap_VerifyWrongOrder(t_ *testing.T) {
	t := newPCMVerifyTest(t_)
	pl := t.NTSDProofList[1:]
	pl = append(pl, t.NTSDProofList[:1]...)
	err := t.PCM.Verify(t.SrcUID, 3, 0, t.BD, pl)
	t.Error(err)
}

func TestProofContextMap_VerifyMalformedProof(t_ *testing.T) {
	t := newPCMVerifyTest(t_)
	var pl ntsdProofList
	pl = append(pl, t.NTSDProofList...)
	pl[0] = append([]byte(nil), t.NTSDProofList[0][1:]...)
	err := t.PCM.Verify(t.SrcUID, 3, 0, t.BD, pl)
	t.Error(err)
}

func TestProofContextMap_Update(t_ *testing.T) {
	const count = 4
	assert := assert.New(t_)

	pcm := newPCMTest(t_).PCM
	pcs, _, _, _ := newKeys(t_, count, "eth", "icon")
	view := &testStateView{
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

	pc0, err := pcm2.ProofContextFor(1)
	assert.NoError(err)
	assert.EqualValues(pcs[0].Bytes(), pc0.Bytes())

	pc1, err := pcm2.ProofContextFor(2)
	assert.NoError(err)
	assert.EqualValues(pcs[1].Bytes(), pc1.Bytes())
}

func TestProofContextMap_UpdateInactivated(t_ *testing.T) {
	const count = 4
	assert := assert.New(t_)

	pcm := newPCMTest(t_).PCM
	pcs, _, _, _ := newKeys(t_, count, "eth", "icon")
	view := &testStateView{
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
				openNetworkIDs:       []int64{},
			},
		},
	}
	builder := NewSectionBuilder(view)
	builder.EnsureSection(1)
	builder.EnsureSection(2)
	builder.NotifyInactivated(2)
	bs, err := builder.Build()
	assert.NoError(err)
	pcm2, err := pcm.Update(sectionPCMUpdateSource{bs})
	assert.NoError(err)

	pc0, err := pcm2.ProofContextFor(1)
	assert.NoError(err)
	assert.EqualValues(pcs[0].Bytes(), pc0.Bytes())

	_, err = pcm2.ProofContextFor(2)
	assert.Error(err)
}
