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
	"math/bits"

	"github.com/icon-project/goloop/module"
)

const hashLen = 32

type Module interface {
	UID() string
	AppendHash(out []byte, data []byte) []byte
	DSA() string
	NewProofContextFromBytes(bs []byte) (module.BTPProofContext, error)
	NewProofContext(pubKeys [][]byte) (module.BTPProofContext, error)
	AddressFromPubKey(pubKey []byte) ([]byte, error)
}

type networkTypeModule struct {
	Module
}

func (ntm *networkTypeModule) Hash(data []byte) []byte {
	return ntm.AppendHash(nil, data)
}

func (ntm *networkTypeModule) merkleRoot(data []byte) []byte {
	for len(data) > hashLen {
		i, j := 0, 0
		for ; i < len(data); i, j = i+hashLen*2, j+hashLen {
			if i+hashLen*2 <= len(data) {
				ntm.AppendHash(data[:j], data[i:i+hashLen*2])
			} else {
				copy(data[j:j+hashLen], data[i:i+hashLen])
			}
		}
		data = data[:j]
	}
	return data[:hashLen]
}

func (ntm *networkTypeModule) MerkleRoot(data module.BytesList) []byte {
	if data.Len() == 0 {
		return nil
	}
	if data.Len() == 1 {
		return data.Get(0)
	}
	dataBuf := make([]byte, 0, data.Len()*hashLen)
	for i := 0; i < data.Len(); i++ {
		dataBuf = append(dataBuf, data.Get(i)...)
	}
	return ntm.merkleRoot(dataBuf)
}

func (ntm *networkTypeModule) merkleProof(data []byte, idx int) []module.MerkleNode {
	proof := make([]module.MerkleNode, 0, bits.Len(uint(len(data))))
	for len(data) > hashLen {
		i, j := 0, 0
		for ; i < len(data); i, j = i+hashLen*2, j+hashLen {
			if i+hashLen*2 <= len(data) {
				var val []byte
				if idx == i {
					val = append(val, data[i+hashLen:i+hashLen*2]...)
					proof = append(
						proof,
						module.MerkleNode{Dir: module.DirRight, Value: val},
					)
					idx = j
				} else if idx == i+hashLen {
					val = append(val, data[i:i+hashLen]...)
					proof = append(
						proof,
						module.MerkleNode{Dir: module.DirLeft, Value: val},
					)
					idx = j
				}
				ntm.AppendHash(data[:j], data[i:i+hashLen*2])
			} else {
				if idx == i {
					idx = j
				}
				copy(data[j:j+hashLen], data[i:i+hashLen])
			}
		}
		data = data[:j]
	}
	return proof
}

func (ntm *networkTypeModule) MerkleProof(data module.BytesList, idx int) []module.MerkleNode {
	if data.Len() == 0 {
		return nil
	}
	if data.Len() == 1 {
		return []module.MerkleNode{}
	}
	dataBuf := make([]byte, 0, data.Len()*hashLen)
	for i := 0; i < data.Len(); i++ {
		dataBuf = append(dataBuf, data.Get(i)...)
	}
	return ntm.merkleProof(dataBuf, idx*hashLen)
}

var modules = make(map[string]module.NetworkTypeModule)

func ForUID(uid string) module.NetworkTypeModule {
	return modules[uid]
}

func register(uid string, mod Module) {
	modules[uid] = &networkTypeModule{Module: mod}
}
