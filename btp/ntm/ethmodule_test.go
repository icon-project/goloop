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

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
)

func rlpListOf(s ...interface{}) []byte {
	var bs []byte
	e := codec.NewEncoderBytes(&bs)
	_ = e.EncodeListOf(s...)
	return bs
}

func keccak256OfRLPList(s ...interface{}) []byte {
	return keccak256(rlpListOf(s...))
}

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

type simpleHasher struct {
	hash []byte
}

func (h simpleHasher) Hash() []byte {
	return h.hash
}

func toHashers(data [][]byte) []interface{ Hash() []byte } {
	hashers := make([]interface{ Hash() []byte }, 0, len(data))
	for _, d := range data {
		hashers = append(hashers, simpleHasher{d})
	}
	return hashers
}

type bytesList [][]byte

func (b bytesList) Len() int {
	return len(b)
}

func (b bytesList) Get(i int) []byte {
	return b[i]
}

func TestEthModule_MerkleRoot(t *testing.T) {
	assert := assert.New(t)
	testCase := []struct {
		exp []byte
		in  [][]byte
	}{
		{
			[]byte{1},
			[][]byte{{1}},
		},
		{
			keccak256OfRLPList(1, 2),
			[][]byte{{1}, {2}},
		},
		{
			keccak256OfRLPList(
				keccak256OfRLPList(1, 2),
				keccak256OfRLPList(3, nil),
			),
			[][]byte{{1}, {2}, {3}},
		},
		{
			keccak256OfRLPList(
				keccak256OfRLPList(1, 2),
				keccak256OfRLPList(3, 4),
			),
			[][]byte{{1}, {2}, {3}, {4}},
		},
		{
			keccak256OfRLPList(
				keccak256OfRLPList(
					keccak256OfRLPList(1, 2),
					keccak256OfRLPList(3, 4),
				),
				keccak256OfRLPList(
					keccak256OfRLPList(5, nil),
					nil,
				),
			),
			[][]byte{{1}, {2}, {3}, {4}, {5}},
		},
		{
			keccak256OfRLPList(
				keccak256OfRLPList(
					keccak256OfRLPList(1, 2),
					keccak256OfRLPList(3, 4),
				),
				keccak256OfRLPList(
					keccak256OfRLPList(5, 6),
					nil,
				),
			),
			[][]byte{{1}, {2}, {3}, {4}, {5}, {6}},
		},
		{
			keccak256OfRLPList(
				keccak256OfRLPList(
					keccak256OfRLPList(1, 2),
					keccak256OfRLPList(3, 4),
				),
				keccak256OfRLPList(
					keccak256OfRLPList(5, 6),
					keccak256OfRLPList(7, nil),
				),
			),
			[][]byte{{1}, {2}, {3}, {4}, {5}, {6}, {7}},
		},
		{
			keccak256OfRLPList(
				keccak256OfRLPList(
					keccak256OfRLPList(1, 2),
					keccak256OfRLPList(3, 4),
				),
				keccak256OfRLPList(
					keccak256OfRLPList(5, 6),
					keccak256OfRLPList(7, 8),
				),
			),
			[][]byte{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}},
		},
	}
	mod := ForUID(ethUID)
	for _, c := range testCase {
		assert.EqualValues(c.exp, mod.MerkleRoot(bytesList(c.in)), "in=%x", c.in)
	}
}
