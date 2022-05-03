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

func init() {
	register(ethUID, &ethModule{})
}
