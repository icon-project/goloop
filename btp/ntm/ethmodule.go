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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const (
	ethUID        = "eth"
	ethAddressLen = 20

	ethBytesByHash = "e" + db.BytesByHash
	ethListByRoot  = "e" + db.ListByMerkleRootBase
)

func appendKeccak256(out []byte, data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(out)
}

func keccak256(data ...[]byte) []byte {
	return appendKeccak256(nil, data...)
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

var ethModuleInstance *networkTypeModule

type ethModuleCore struct{}

func (m *ethModuleCore) UID() string {
	return ethUID
}

func (m *ethModuleCore) AppendHash(out []byte, data []byte) []byte {
	return appendKeccak256(out, data)
}

func (m *ethModuleCore) DSAModule() module.DSAModule {
	return secp256k1DSAModuleInstance
}

func (m *ethModuleCore) NewProofContextFromBytes(bs []byte) (proofContextCore, error) {
	return newSecp256k1ProofContextFromBytes(ethModuleInstance, bs)
}

func (m *ethModuleCore) NewProofContext(keys [][]byte) (proofContextCore, error) {
	return newSecp256k1ProofContext(ethModuleInstance, keys)
}

func (m *ethModuleCore) AddressFromPubKey(pubKey []byte) ([]byte, error) {
	return newEthAddressFromPubKey(pubKey)
}

func (m *ethModuleCore) BytesByHashBucket() db.BucketID {
	return ethBytesByHash
}

func (m *ethModuleCore) ListByMerkleRootBucket() db.BucketID {
	return ethListByRoot
}

func (m *ethModuleCore) NewProofFromBytes(bs []byte) (module.BTPProof, error) {
	return newSecp256k1ProofFromBytes(bs)
}

func (m *ethModuleCore) NetworkTypeKeyFromDSAKey(key []byte) ([]byte, error) {
	return key, nil
}

func init() {
	ethModuleInstance = register(ethUID, &ethModuleCore{})
}
