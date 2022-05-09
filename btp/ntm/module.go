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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type Module interface {
	UID() string
	Hash(data []byte) []byte
	DSA() string
	NewProofContextFromBytes(bs []byte) (module.BTPProofContext, error)
	NewProofContext(pubKeys [][]byte) (module.BTPProofContext, error)
	AddressFromPubKey(pubKey []byte) ([]byte, error)
}

type networkTypeModule struct {
	Module
}

func (ntm *networkTypeModule) merkleRoot(data [][]byte) []byte {
	encoderBuf := make([]byte, 0, 128)
	for len(data) > 1 {
		i, j := 0, 0
		for ; i < len(data); i, j = i+2, j+1 {
			if i+1 < len(data) {
				e := codec.NewEncoderBytes(&encoderBuf)
				log.Must(e.EncodeListOf(data[i], data[i+1]))
				data[j] = ntm.Hash(encoderBuf)
			} else {
				data[j] = data[i]
			}
		}
		data = data[:j]
	}
	return data[0]
}

func (ntm *networkTypeModule) MerkleRoot(data module.BytesList) []byte {
	if data.Len() == 0 {
		return nil
	}
	if data.Len() == 1 {
		return data.Get(0)
	}
	dataBuf := make([][]byte, 0, data.Len())
	for i := 0; i < data.Len(); i++ {
		dataBuf = append(dataBuf, data.Get(i))
	}
	return ntm.merkleRoot(dataBuf)
}

var modules = make(map[string]module.NetworkTypeModule)

func ForUID(uid string) module.NetworkTypeModule {
	return modules[uid]
}

func register(uid string, mod Module) {
	modules[uid] = &networkTypeModule{Module: mod}
}
