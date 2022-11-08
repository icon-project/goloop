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

	"github.com/icon-project/goloop/common/wallet"
)

func testModuleBasics(t *testing.T, uid, dsa string) {
	assert := assert.New(t)
	const count = 4
	mod := ForUID(uid)
	assert.EqualValues(uid, mod.UID())
	assert.EqualValues(dsa, mod.DSA())

	addrs := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		w := wallet.New()
		addr, err := mod.AddressFromPubKey(w.PublicKey())
		assert.NoError(err)
		addrs = append(addrs, addr)
	}
	ctx, err := mod.NewProofContext(addrs)
	assert.NoError(err)
	assert.EqualValues(uid, ctx.UID())
	assert.EqualValues(dsa, ctx.DSA())
	assert.EqualValues(mod.Hash(ctx.Bytes()), ctx.Hash())
	ctxBytes := ctx.Bytes()
	ctx2, err := mod.NewProofContextFromBytes(ctxBytes)
	assert.NoError(err)
	assert.EqualValues(ctx.Bytes(), ctx2.Bytes())
}

func TestModule_Basics(t *testing.T) {
	InitIconModule()
	testModuleBasics(t, "eth", "ecdsa/secp256k1")
	testModuleBasics(t, "icon", "ecdsa/secp256k1")
}
