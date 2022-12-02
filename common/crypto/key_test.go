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
	"encoding/hex"
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

func TestParsePrivateKey(t *testing.T) {
	sk, _ := GenerateKeyPair()
	bs := sk.Bytes()
	bss := hex.EncodeToString(bs)

	fmt.Println("PrivateKey:", bss)
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		args    args
		bytes   string
		pubkeyC string
		pubkeyU string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "InvalidKeyBytes1",
			args:    args{"3c28a9e536e73cda915aa19e58fcaf6ecb997204959797ed4dc4361cbbd2"},
			wantErr: assert.Error,
		},
		{
			name:    "InvalidKeyBytes2",
			args:    args{"3c28a9e536e73cda915aa19e58fcaf6ecb997204959797ed4dc4361cbbd228e278"},
			wantErr: assert.Error,
		},
		{
			name:    "ValidKeyBytes1",
			args:    args{"3c28a9e536e73cda915aa19e58fcaf6ecb997204959797ed4dc4361cbbd228e2"},
			wantErr: assert.NoError,
			bytes:   "3c28a9e536e73cda915aa19e58fcaf6ecb997204959797ed4dc4361cbbd228e2",
			pubkeyC: "0245c006e811c8afc94f1042d1c7991ec218cd05532987125a7cbe79523f687ac9",
			pubkeyU: "0445c006e811c8afc94f1042d1c7991ec218cd05532987125a7cbe79523f687ac9478b846fa60379cb42a964729da8a1a4ba9f1a8a7b212067ebaf45963ba20c2e",
		},
		{
			name:    "ValidKeyBytes2",
			args:    args{"3a0cb098f0377e96ac5b5161de86020790235187461ab37137b15d647586d028"},
			wantErr: assert.NoError,
			bytes:   "3a0cb098f0377e96ac5b5161de86020790235187461ab37137b15d647586d028",
			pubkeyC: "021a2e95038c5d54d1539231b4e562d399ee4a4c64fb7f3019d98e1a08352ede7d",
			pubkeyU: "041a2e95038c5d54d1539231b4e562d399ee4a4c64fb7f3019d98e1a08352ede7db54c1ae0a7d84e02010f8a19ea2f9925c7d7252f10b91d11486dda5678a0a4f6",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skBytes, err := hex.DecodeString(tt.args.key)
			assert.NoError(t, err)
			sk, err := ParsePrivateKey(skBytes)
			if !tt.wantErr(t, err, fmt.Sprintf("ParsePrivateKey(%v)", tt.args.key)) {
				return
			}
			if err != nil {
				return
			}
			assert.Equalf(t, tt.bytes, hex.EncodeToString(sk.Bytes()), "ParsePrivateKey(%v).Bytes()", tt.bytes)

			pk := sk.PublicKey()
			assert.Equalf(t, tt.pubkeyC, hex.EncodeToString(pk.SerializeCompressed()), "ParsePrivateKey(%v).PublicKey().SerializeCompressed()", tt.bytes)
			assert.Equalf(t, tt.pubkeyU, hex.EncodeToString(pk.SerializeUncompressed()), "ParsePrivateKey(%v).PublicKey().SerializeUncompressed()", tt.bytes)
		})
	}
}
