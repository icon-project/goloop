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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/wallet"
)

func TestSecp256k1DSAModule_Verify(t *testing.T) {
	assert := assert.New(t)

	w := wallet.New()
	dsam := DSAModuleForName(secp256k1DSA)
	assert.NoError(dsam.Verify(w.PublicKey()))

	pkBytes := w.PublicKey()
	assert.Error(dsam.Verify(pkBytes[:len(pkBytes)-1]))

	_, pk := crypto.GenerateKeyPair()
	pkBytes = pk.SerializeUncompressed()
	assert.NoError(dsam.Verify(pkBytes))

	assert.Error(dsam.Verify(pkBytes[:len(pkBytes)-1]))
}
