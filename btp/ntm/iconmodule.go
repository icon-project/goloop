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

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

// CAUTION:
// ICON MODULE IS JUST FOR TEST PURPOSE. In product, please USE ETH MODULE since
// icon score can easily verify eth proof.

const (
	iconUID          = "icon"
	iconAddressIDLen = 20

	iconBytesByHash = "i" + db.BytesByHash
	iconListByRoot  = "i" + db.ListByMerkleRootBase
)

func newIconAddressFromPubKey(pubKey []byte) ([]byte, error) {
	if len(pubKey) == crypto.PublicKeyLenCompressed {
		pk, err := crypto.ParsePublicKey(pubKey)
		if err != nil {
			return nil, err
		}
		pubKey = pk.SerializeUncompressed()
	}
	digest := crypto.SHA3Sum256(pubKey[1:])
	return common.NewAccountAddress(digest[len(digest)-iconAddressIDLen:]).Bytes(), nil
}

var iconModuleInstance *networkTypeModule

type iconModuleCore struct{}

func (m *iconModuleCore) UID() string {
	return iconUID
}

func (m *iconModuleCore) AppendHash(out []byte, data []byte) []byte {
	h := sha3.New256()
	h.Write(data)
	return h.Sum(out)
}

func (m *iconModuleCore) DSAModule() module.DSAModule {
	return secp256k1DSAModuleInstance
}

func (m *iconModuleCore) NewProofContextFromBytes(bs []byte) (proofContextCore, error) {
	return newSecp256k1ProofContextFromBytes(iconModuleInstance, bs)
}

func (m *iconModuleCore) NewProofContext(keys [][]byte) (proofContextCore, error) {
	return newSecp256k1ProofContext(iconModuleInstance, keys)
}

func (m *iconModuleCore) AddressFromPubKey(pubKey []byte) ([]byte, error) {
	return newIconAddressFromPubKey(pubKey)
}

func (m *iconModuleCore) BytesByHashBucket() db.BucketID {
	return iconBytesByHash
}

func (m *iconModuleCore) ListByMerkleRootBucket() db.BucketID {
	return iconListByRoot
}

func (m *iconModuleCore) NewProofFromBytes(bs []byte) (module.BTPProof, error) {
	return newSecp256k1ProofFromBytes(bs)
}

func (m *iconModuleCore) NetworkTypeKeyFromDSAKey(key []byte) ([]byte, error) {
	return key, nil
}

func init() {
	iconModuleInstance = register(iconUID, &iconModuleCore{})
}
