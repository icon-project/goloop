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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignature_Uninitialized(t *testing.T) {
	var sig Signature
	t.Run("SerializeRS", func(t *testing.T) {
		bs, err := sig.SerializeRS()
		assert.Error(t, err)
		assert.Nil(t, bs)
	})

	t.Run("SerializeRSV", func(t *testing.T) {
		bs, err := sig.SerializeRSV()
		assert.Error(t, err)
		assert.Nil(t, bs)
	})

	t.Run("SerializeVRS", func(t *testing.T) {
		bs, err := sig.SerializeRSV()
		assert.Error(t, err)
		assert.Nil(t, bs)
	})

	t.Run("HasV", func(t *testing.T) {
		assert.False(t, sig.HasV())
	})
}

func TestSignature_NewSignature(t *testing.T) {
	sk, _ := GenerateKeyPair()
	hash := SHA3Sum256([]byte("TEST Data"))
	t.Run("NilParameter", func(t *testing.T) {
		sig, err := NewSignature(nil, sk)
		assert.Error(t, err)
		assert.Nil(t, sig)

		sig, err = NewSignature(hash, nil)
		assert.Error(t, err)
		assert.Nil(t, sig)

		sig, err = NewSignature(nil, nil)
		assert.Error(t, err)
		assert.Nil(t, sig)
	})

	t.Run("ValidParameter", func(t *testing.T) {
		sig, err := NewSignature(hash, sk)
		assert.NoError(t, err)
		assert.NotNil(t, sig)
	})
}

func TestSignature_Serialize(t *testing.T) {
	sk, pk := GenerateKeyPair()
	assert.NotNil(t, sk)
	assert.NotNil(t, pk)

	data := []byte("test data")
	hash := SHA3Sum256(data)

	sig, err := NewSignature(hash, sk)
	assert.NoError(t, err)
	assert.True(t, sig.HasV())

	rsv, err := sig.SerializeRSV()
	assert.NoError(t, err)
	assert.Len(t, rsv, SignatureLenRawWithV)

	t.Run("SerializeRS", func(t *testing.T) {
		rs, err := sig.SerializeRS()
		assert.NoError(t, err)
		assert.Len(t, rs, SignatureLenRaw)
		assert.Equal(t, rsv[0:SignatureLenRaw], rs)

		sig2, err := ParseSignature(rs)
		assert.NoError(t, err)
		assert.False(t, sig2.HasV())

		rs2, err := sig2.SerializeRS()
		assert.NoError(t, err)
		assert.Equal(t, rs, rs2)
	})

	t.Run("SerializeVRS", func(t *testing.T) {
		vrs, err := sig.SerializeVRS()
		assert.NoError(t, err)
		assert.NotNil(t, vrs)
		assert.Len(t, vrs, SignatureLenRawWithV)
		assert.Equal(t, rsv[0:SignatureLenRaw], vrs[1:])
		assert.Equal(t, rsv[SignatureLenRaw:], vrs[:1])

		sig2, err := ParseSignatureVRS(vrs)
		assert.NoError(t, err)
		assert.True(t, sig2.HasV())

		vrs2, err := sig2.SerializeVRS()
		assert.NoError(t, err)
		assert.Equal(t, vrs, vrs2)
	})
}

func Test_ParseSignature(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		sig, err := ParseSignature(nil)
		assert.Error(t, err)
		assert.Nil(t, sig)
	})

	t.Run("Less", func(t *testing.T) {
		sig, err := ParseSignature([]byte{1, 2, 3})
		assert.Error(t, err)
		assert.Nil(t, sig)
	})

	t.Run("RSSize", func(t *testing.T) {
		rs := make([]byte, SignatureLenRaw)
		sig, err := ParseSignature(rs)
		assert.NoError(t, err)
		assert.NotNil(t, sig)
		assert.False(t, sig.HasV())
	})

	t.Run("RSVSize", func(t *testing.T) {
		rsv := make([]byte, SignatureLenRawWithV)
		sig, err := ParseSignature(rsv)
		assert.NoError(t, err)
		assert.NotNil(t, sig)
		assert.True(t, sig.HasV())
	})

	t.Run("RSVSize+1", func(t *testing.T) {
		bad := make([]byte, SignatureLenRawWithV+1)
		sig, err := ParseSignature(bad)
		assert.Error(t, err)
		assert.Nil(t, sig)
	})
}

func Test_ParseSignatureVRS(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		sig, err := ParseSignatureVRS(nil)
		assert.Error(t, err)
		assert.Nil(t, sig)
	})

	t.Run("Less", func(t *testing.T) {
		sig, err := ParseSignatureVRS([]byte{1, 2, 3})
		assert.Error(t, err)
		assert.Nil(t, sig)
	})

	t.Run("RSSize", func(t *testing.T) {
		rs := make([]byte, SignatureLenRaw)
		sig, err := ParseSignatureVRS(rs)
		assert.Error(t, err)
		assert.Nil(t, sig)
	})

	t.Run("VRSSize", func(t *testing.T) {
		rsv := make([]byte, SignatureLenRawWithV)
		sig, err := ParseSignatureVRS(rsv)
		assert.NoError(t, err)
		assert.NotNil(t, sig)
		assert.True(t, sig.HasV())
	})

	t.Run("VRSSize+1", func(t *testing.T) {
		rs := make([]byte, SignatureLenRawWithV+1)
		sig, err := ParseSignatureVRS(rs)
		assert.Error(t, err)
		assert.Nil(t, sig)
	})
}
