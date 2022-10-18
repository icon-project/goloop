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
	"github.com/icon-project/goloop/common/crypto"
)

const (
	secp256k1DSA = "ecdsa/secp256k1"
)

type secp256k1DSAModule struct {
}

func (s secp256k1DSAModule) Name() string {
	return secp256k1DSA
}

func (s secp256k1DSAModule) Verify(pubKey []byte) error {
	_, err := crypto.ParsePublicKey(pubKey)
	return err
}

var secp256k1DSAModuleInstance secp256k1DSAModule

func init() {
	registerDSAModule(secp256k1DSAModuleInstance)
}
