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

	"github.com/icon-project/goloop/common/crypto"
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

func newEthAddressFromPubKey(pubKey []byte) ([]byte, error) {
	if len(pubKey) == crypto.PublicKeyLenCompressed {
		pk, err := crypto.ParsePublicKey(pubKey)
		if err != nil {
			return nil, err
		}
		pubKey = pk.SerializeUncompressed()
	}
	digest := keccak256(pubKey[1:])
	return digest[len(digest)-ethAddressLen:], nil
}

var ethModuleInstance ethModule

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

func (m *ethModule) NewProofContext(keys [][]byte) (module.BTPProofContext, error) {
	return newEthProofContext(keys)
}

func (m *ethModule) AddressFromPubKey(pubKey []byte) ([]byte, error) {
	return newEthAddressFromPubKey(pubKey)
}

func init() {
	register(ethUID, &ethModule{})
}
