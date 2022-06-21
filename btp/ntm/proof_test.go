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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/wallet"
	"github.com/icon-project/goloop/module"
)

type testSetup struct {
	assert  *assert.Assertions
	count   int
	wallets []*walletProvider
	pubKeys [][]byte
	addrs   [][]byte
	pc      module.BTPProofContext
}

type walletProvider struct {
	wallets map[string]module.BaseWallet
}

func (w walletProvider) WalletFor(keyType string) module.BaseWallet {
	return w.wallets[keyType]
}

func newSecp256k1WalletProvider() (*walletProvider, module.Wallet) {
	w := wallet.New()
	wp := walletProvider{
		wallets: map[string]module.BaseWallet{
			secp256k1DSA: w,
		},
	}
	return &wp, w
}

func newEthTestSetup(t *testing.T, count int) *testSetup {
	return newTestSetup(t, ethModuleInstance, count)
}

func newTestSetup(t *testing.T, mod *networkTypeModule, count int) *testSetup {
	s := &testSetup{
		assert:  assert.New(t),
		count:   count,
		wallets: make([]*walletProvider, 0, count),
		pubKeys: make([][]byte, 0, count),
		addrs:   make([][]byte, 0, count),
	}
	for i := 0; i < count; i++ {
		wp, w := newSecp256k1WalletProvider()
		s.wallets = append(s.wallets, wp)
		s.pubKeys = append(s.pubKeys, w.PublicKey())
		addr, err := newEthAddressFromPubKey(s.pubKeys[i])
		s.assert.NoError(err)
		s.addrs = append(s.addrs, addr)
	}
	s.pc = mod.NewProofContext(s.addrs)
	return s
}

func (s *testSetup) newProofOfLen(l int, msgHash []byte) module.BTPProof {
	p := s.pc.NewProof()
	for i := 0; i < l; i++ {
		pp, err := s.pc.NewProofPart(msgHash, s.wallets[i])
		s.assert.NoError(err)
		p.Add(pp)
	}
	return p
}

func TestEthProofContext_NewProofPart_OK(t *testing.T) {
	s := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	for i := 0; i < s.count; i++ {
		pp, err := s.pc.NewProofPart(msgHash, s.wallets[i])
		s.assert.NoError(err)
		_, err = s.pc.VerifyPart(msgHash, pp)
		s.assert.NoError(err)
	}
}

func TestEthProofContext_NewProofPart_FailInvalidPK(t *testing.T) {
	s := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	wp, _ := newSecp256k1WalletProvider()
	_, err := s.pc.NewProofPart(msgHash, wp)
	s.assert.Error(err)
}

func TestEthProofContext_VerifyPart_FailWrongMessage(t *testing.T) {
	s := newEthTestSetup(t, 4)
	pp, err := s.pc.NewProofPart(keccak256([]byte("abc")), s.wallets[0])
	s.assert.NoError(err)
	_, err = s.pc.VerifyPart(keccak256([]byte("abcd")), pp)
	s.assert.Error(err)
}

func TestEthProofPart_codec(t *testing.T) {
	s := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	pp, err := s.pc.NewProofPart(msgHash, s.wallets[2])
	s.assert.NoError(err)
	epp := pp.(*secp256k1ProofPart)
	ppBytes := codec.MustMarshalToBytes(epp)
	s.assert.EqualValues(ppBytes, codec.MustMarshalToBytes(pp))
	s.assert.EqualValues(ppBytes, pp.Bytes())
	var epp2 secp256k1ProofPart
	codec.MustUnmarshalFromBytes(ppBytes, &epp2)
	_, err = s.pc.VerifyPart(msgHash, &epp2)
	s.assert.NoError(err)
}

func TestEthProof_codec(t *testing.T) {
	s := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	p := s.newProofOfLen(3, msgHash)
	ep := p.(*secp256k1Proof)
	pBytes := codec.MustMarshalToBytes(ep)
	s.assert.EqualValues(pBytes, codec.MustMarshalToBytes(p))
	s.assert.EqualValues(pBytes, p.Bytes())
	var ep2 secp256k1Proof
	codec.MustUnmarshalFromBytes(pBytes, &ep2)
	s.assert.NoError(s.pc.Verify(msgHash, &ep2))
}

func TestEthProofContext_codec(t *testing.T) {
	s := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	p := s.newProofOfLen(3, msgHash)
	pcBytes := s.pc.Bytes()
	pc2, err := ethModuleInstance.NewProofContextFromBytes(pcBytes)
	s.assert.NoError(err)
	s.assert.NoError(pc2.Verify(msgHash, p))
	s.pc = pc2
	p2 := s.newProofOfLen(3, msgHash)
	s.assert.NoError(s.pc.Verify(msgHash, p2))
}

func TestEthProofContext_Verify(t *testing.T) {
	msgHash := keccak256([]byte("abc"))
	testCase := []struct {
		ok      bool
		ppCount int
		pkCount int
	}{
		{false, 0, 1},
		{true, 1, 1},

		{false, 1, 2},
		{true, 2, 2},

		{false, 2, 3},
		{true, 3, 3},

		{false, 2, 4},
		{true, 3, 4},

		{false, 3, 5},
		{true, 4, 5},

		{false, 4, 6},
		{true, 5, 6},

		{false, 4, 7},
		{true, 5, 7},
	}
	for _, c := range testCase {
		s := newEthTestSetup(t, c.pkCount)
		p := s.newProofOfLen(c.ppCount, msgHash)
		err := s.pc.Verify(msgHash, p)
		if c.ok {
			s.assert.NoError(err, "Verify exp=%v ppCount=%d pkCount=%d", c.ok, c.ppCount, c.pkCount)
		} else {
			s.assert.Error(err, "Verify exp=%v ppCount=%d pkCount=%d", c.ok, c.ppCount, c.pkCount)
		}
		p2, err := s.pc.NewProofFromBytes(p.Bytes())
		s.assert.NoError(err)
		err = s.pc.Verify(msgHash, p2)
		if c.ok {
			s.assert.NoError(err, "VerifyByProofBytes exp=%v ppCount=%d pkCount=%d bytes=%x", c.ok, c.ppCount, c.pkCount, p.Bytes())
		} else {
			s.assert.Error(err, "VerifyByProofBytes exp=%v ppCount=%d pkCount=%d bytes=%x", c.ok, c.ppCount, c.pkCount, p.Bytes())
		}
	}
}

func TestEthProofContext_Verify_FailInvalidPK(t *testing.T) {
	s := newEthTestSetup(t, 4)
	s2 := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	p := s.newProofOfLen(2, msgHash)
	pp, err := s2.pc.NewProofPart(msgHash, s2.wallets[0])
	s.assert.NoError(err)
	p.Add(pp)
	s.assert.Error(s.pc.Verify(msgHash, p))
}

func TestEthProofContext_Verify_FailDuplicatedPK(t *testing.T) {
	s := newEthTestSetup(t, 4)
	msgHash := keccak256([]byte("abc"))
	p := s.newProofOfLen(2, msgHash)
	pp, err := s.pc.NewProofPart(msgHash, s.wallets[0])
	s.assert.NoError(err)
	p.Add(pp)
	s.assert.Error(s.pc.Verify(msgHash, p))
}
