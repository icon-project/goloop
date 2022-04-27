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
	"golang.org/x/crypto/sha3"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

const (
	ethUID        = "eth"
	ethDSA        = "ecdsa/secp256k1"
	ethAddressLen = 20
)

func keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

type ethAddress []byte

func newEthAddressFromPubKey(pubKey []byte) ethAddress {
	if len(pubKey) == crypto.PublicKeyLenCompressed {
		pk, err := crypto.ParsePublicKey(pubKey)
		if err != nil {
			log.Panicf("%+v", err)
		}
		pubKey = pk.SerializeUncompressed()
	}
	digest := keccak256(pubKey[1:])
	return digest[len(digest)-ethAddressLen:]
}

type ethModule struct{}

func (m *ethModule) UID() string {
	return ethUID
}

func (m *ethModule) Hash(data []byte) []byte {
	return keccak256(data)
}

func (m *ethModule) DSA() string {
	return ethDSA
}

func (m *ethModule) NewProofContextFromBytes(bs []byte) (module.BTPProofContext, error) {
	return newEthProofContextFromBytes(bs)
}

func (m *ethModule) NewProofContext(pubKeys [][]byte) module.BTPProofContext {
	return newEthProofContext(pubKeys)
}

func (m *ethModule) merkleRoot(data [][]byte) []byte {
	encoderBuf := make([]byte, 0, 128)
	for len(data) > 1 {
		if len(data)%2 != 0 {
			data = append(data, nil)
		}
		i, j := 0, 0
		for ; i < len(data); i, j = i+2, j+1 {
			e := codec.NewEncoderBytes(&encoderBuf)
			log.Must(e.EncodeListOf(data[i], data[i+1]))
			data[j] = keccak256(encoderBuf)
		}
		data = data[:j]
	}
	return data[0]
}

func (m *ethModule) MerkleRoot(data [][]byte) []byte {
	if len(data) == 0 {
		return nil
	}
	if len(data) == 1 {
		return data[0]
	}
	evenedLen := (len(data) + 1) &^ 1
	dataBuf := make([][]byte, len(data), evenedLen)
	copy(dataBuf, data)
	return m.merkleRoot(dataBuf)
}

func (m *ethModule) MerkleRootHashers(hashers []interface{ Hash() []byte }) []byte {
	evenedLen := (len(hashers) + 1) &^ 1
	data := make([][]byte, 0, evenedLen)
	for _, hasher := range hashers {
		data = append(data, hasher.Hash())
	}
	return m.merkleRoot(data)
}

func (m *ethModule) MerkleRootHashCat(hashes []byte) []byte {
	//TODO implement me
	panic("implement me")
}

func init() {
	register(ethUID, &ethModule{})
}
