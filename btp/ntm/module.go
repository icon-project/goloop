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

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

const hashLen = 32

type moduleCore interface {
	UID() string
	AppendHash(out []byte, data []byte) []byte
	DSAModule() module.DSAModule
	NewProofContextFromBytes(bs []byte) (proofContextCore, error)
	NewProofContext(pubKeys [][]byte) proofContextCore
	AddressFromPubKey(pubKey []byte) ([]byte, error)
	BytesByHashBucket() db.BucketID
	ListByMerkleRootBucket() db.BucketID
	NewProofFromBytes(bs []byte) (module.BTPProof, error)
	NetworkTypeKeyFromDSAKey(key []byte) ([]byte, error)
}

type networkTypeModule struct {
	core moduleCore
}

func (ntm *networkTypeModule) UID() string {
	return ntm.core.UID()
}

func (ntm *networkTypeModule) AppendHash(out []byte, data []byte) []byte {
	return ntm.core.AppendHash(out, data)
}

func (ntm *networkTypeModule) DSA() string {
	return ntm.core.DSAModule().Name()
}

func (ntm *networkTypeModule) NewProofContextFromBytes(bs []byte) (module.BTPProofContext, error) {
	pcCore, err := ntm.core.NewProofContextFromBytes(bs)
	if err != nil {
		return nil, err
	}
	return &proofContext{
		core: pcCore,
	}, nil
}

func (ntm *networkTypeModule) NewProofContext(keys [][]byte) module.BTPProofContext {
	return &proofContext{
		core: ntm.core.NewProofContext(keys),
	}
}

func (ntm *networkTypeModule) AddressFromPubKey(pubKey []byte) ([]byte, error) {
	return ntm.core.AddressFromPubKey(pubKey)
}

func (ntm *networkTypeModule) Hash(data []byte) []byte {
	return ntm.core.AppendHash(nil, data)
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
					proof = append(
						proof,
						module.MerkleNode{Dir: module.DirRight, Value: nil},
					)
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

func (ntm *networkTypeModule) BytesByHashBucket() db.BucketID {
	return ntm.core.BytesByHashBucket()
}

func (ntm *networkTypeModule) ListByMerkleRootBucket() db.BucketID {
	return ntm.core.ListByMerkleRootBucket()
}

func (ntm *networkTypeModule) NewProofFromBytes(bs []byte) (module.BTPProof, error) {
	return ntm.core.NewProofFromBytes(bs)
}

func (ntm *networkTypeModule) NetworkTypeKeyFromDSAKey(key []byte) ([]byte, error) {
	return ntm.core.NetworkTypeKeyFromDSAKey(key)
}

func (ntm *networkTypeModule) DSAModule() module.DSAModule {
	return ntm.core.DSAModule()
}

var modules = make(map[string]module.NetworkTypeModule)

func Modules() map[string]module.NetworkTypeModule {
	return modules
}

func ForUID(uid string) module.NetworkTypeModule {
	return modules[uid]
}

type simpleHasher struct {
	mod module.NetworkTypeModule
}

func (h simpleHasher) Name() string {
	return h.mod.UID() + "/hash"
}

func (h simpleHasher) Hash(value []byte) []byte {
	return h.mod.Hash(value)
}

type catBytesList []byte

func (l catBytesList) Len() int {
	return len(l) / hashLen
}

func (l catBytesList) Get(i int) []byte {
	return l[i*hashLen : i*hashLen+hashLen]
}

type merkleRootHasher struct {
	mod module.NetworkTypeModule
}

func (h *merkleRootHasher) Name() string {
	return h.mod.UID() + "/merkleRoot"
}

func (h *merkleRootHasher) Hash(value []byte) []byte {
	return h.mod.MerkleRoot(catBytesList(value))
}

func register(uid string, mod moduleCore) *networkTypeModule {
	networkTypeModule := &networkTypeModule{core: mod}
	modules[uid] = networkTypeModule
	db.RegisterHasher(mod.BytesByHashBucket(), &simpleHasher{
		mod: networkTypeModule,
	})
	db.RegisterHasher(mod.ListByMerkleRootBucket(), &merkleRootHasher{
		mod: networkTypeModule,
	})
	return networkTypeModule
}
