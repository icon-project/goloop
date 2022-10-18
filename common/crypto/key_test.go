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

package crypto

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKeyPair(t *testing.T) {
	sk, pk := GenerateKeyPair()
	assert.NotNil(t, sk)
	assert.NotNil(t, pk)
	assert.Equal(t, sk.String(), fmt.Sprint(sk))
	assert.Equal(t, pk.String(), fmt.Sprint(pk))

	pk2 := sk.PublicKey()
	assert.NotNil(t, pk2)
	assert.True(t, pk2.Equal(pk))
}

func TestKeyBytes(t *testing.T) {
	sk, pk := GenerateKeyPair()
	skBytes := sk.Bytes()
	assert.NotNil(t, skBytes)
	assert.Len(t, skBytes, PrivateKeyLen)

	sk2, err := ParsePrivateKey(skBytes)
	assert.NoError(t, err)
	assert.Equal(t, skBytes, sk2.Bytes())

	pkCBytes := pk.SerializeCompressed()
	assert.NotNil(t, pkCBytes)
	assert.Len(t, pkCBytes, PublicKeyLenCompressed)

	pk2, err := ParsePublicKey(pkCBytes)
	assert.NoError(t, err)
	assert.NotNil(t, pk2)
	assert.True(t, pk.Equal(pk2))

	pkUBytes := pk.SerializeUncompressed()
	assert.NotNil(t, pkUBytes)
	assert.Len(t, pkUBytes, PublicKeyLenUncompressed)

	pk3, err := ParsePublicKey(pkCBytes)
	assert.NoError(t, err)
	assert.NotNil(t, pk3)
	assert.True(t, pk.Equal(pk3))
}
