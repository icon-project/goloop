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

import "github.com/icon-project/goloop/module"

// Module represents a network type module.
type Module interface {
	UID() string
	Hash(data []byte) []byte
	DSA() string
	NewProofContextFromBytes(bs []byte) (module.BTPProofContext, error)
	NewProofContext(pubKeys [][]byte) module.BTPProofContext
	MerkleRoot(data [][]byte) []byte
	MerkleRootHashCat(hashes []byte) []byte
}

var modules = make(map[string]Module)

func ForUID(uid string) Module {
	return modules[uid]
}

func register(uid string, ntm Module) {
	modules[uid] = ntm
}
