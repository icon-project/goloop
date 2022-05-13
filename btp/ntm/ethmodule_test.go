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
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
)

func Test_keccak256(t *testing.T) {
	assert := assert.New(t)
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	assert.EqualValues(exp, keccak256(msg))
}

func TestEthModule_newAddressFromPubKey(t *testing.T) {
	assert := assert.New(t)

	skHex := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	skBytes, err := hex.DecodeString(skHex)
	assert.NoError(err)
	sk, err := crypto.ParsePrivateKey(skBytes)
	assert.NoError(err)
	expAddrHex := "970e8128ab834e8eac17ab8e3812f010678cf791"
	expAddrBytes, err := hex.DecodeString(expAddrHex)
	assert.NoError(err)
	pk := sk.PublicKey()
	addr, err := newEthAddressFromPubKey(pk.SerializeUncompressed())
	assert.NoError(err)
	assert.EqualValues(expAddrBytes, addr)
}

func TestEthModule_MerkleRoot(t *testing.T) {
	mod := ForUID(ethUID)
	var h = func(b byte) []byte {
		return mod.Hash([]byte{b})
	}
	assert := assert.New(t)
	testCase := []struct {
		exp []byte
		in  module.BytesSlice
	}{
		{
			h(1),
			[][]byte{h(1)},
		},
		{
			keccak256(h(1), h(2)),
			[][]byte{h(1), h(2)},
		},
		{
			keccak256(
				keccak256(h(1), h(2)),
				h(3),
			),
			[][]byte{h(1), h(2), h(3)},
		},
		{
			keccak256(
				keccak256(h(1), h(2)),
				keccak256(h(3), h(4)),
			),
			[][]byte{h(1), h(2), h(3), h(4)},
		},
		{
			keccak256(
				keccak256(
					keccak256(h(1), h(2)),
					keccak256(h(3), h(4)),
				),
				h(5),
			),
			[][]byte{h(1), h(2), h(3), h(4), h(5)},
		},
		{
			keccak256(
				keccak256(
					keccak256(h(1), h(2)),
					keccak256(h(3), h(4)),
				),
				keccak256(h(5), h(6)),
			),
			[][]byte{h(1), h(2), h(3), h(4), h(5), h(6)},
		},
		{
			keccak256(
				keccak256(
					keccak256(h(1), h(2)),
					keccak256(h(3), h(4)),
				),
				keccak256(
					keccak256(h(5), h(6)),
					h(7),
				),
			),
			[][]byte{h(1), h(2), h(3), h(4), h(5), h(6), h(7)},
		},
		{
			keccak256(
				keccak256(
					keccak256(h(1), h(2)),
					keccak256(h(3), h(4)),
				),
				keccak256(
					keccak256(h(5), h(6)),
					keccak256(h(7), h(8)),
				),
			),
			[][]byte{h(1), h(2), h(3), h(4), h(5), h(6), h(7), h(8)},
		},
	}
	for _, c := range testCase {
		assert.EqualValues(c.exp, mod.MerkleRoot(&c.in), "in=%x", c.in)
	}
}

func TestEthModule_MerkleProof(t *testing.T) {
	mod := ForUID(ethUID)
	var h = func(b byte) []byte {
		return mod.Hash([]byte{b})
	}
	assert := assert.New(t)
	testCase := []struct {
		exp  []module.MerkleNode
		data module.BytesSlice
		idx  int
	}{
		{
			[]module.MerkleNode{},
			[][]byte{h(0)},
			0,
		},
		{
			[]module.MerkleNode{{module.DirRight, h(1)}},
			[][]byte{h(0), h(1)},
			0,
		},
		{
			[]module.MerkleNode{{module.DirLeft, h(0)}},
			[][]byte{h(0), h(1)},
			1,
		},
		{
			[]module.MerkleNode{
				{module.DirRight, h(1)},
				{module.DirRight, h(2)},
			},
			[][]byte{h(0), h(1), h(2)},
			0,
		},
		{
			[]module.MerkleNode{
				{module.DirLeft, h(0)},
				{module.DirRight, h(2)},
			},
			[][]byte{h(0), h(1), h(2)},
			1,
		},
		{
			[]module.MerkleNode{
				{module.DirRight, nil},
				{module.DirLeft, keccak256(h(0), h(1))},
			},
			[][]byte{h(0), h(1), h(2)},
			2,
		},
		{
			[]module.MerkleNode{
				{module.DirRight, h(1)},
				{module.DirRight, keccak256(h(2), h(3))},
				{module.DirRight, h(4)},
			},
			[][]byte{h(0), h(1), h(2), h(3), h(4)},
			0,
		},
		{
			[]module.MerkleNode{
				{module.DirRight, nil},
				{module.DirRight, nil},
				{
					module.DirLeft, keccak256(
						keccak256(h(0), h(1)),
						keccak256(h(2), h(3)),
					),
				},
			},
			[][]byte{h(0), h(1), h(2), h(3), h(4)},
			4,
		},
	}
	for i, c := range testCase {
		assert.EqualValues(c.exp, mod.MerkleProof(&c.data, c.idx), "case=%d data=%x idx=%d", i, c.data, c.idx)
	}
}
