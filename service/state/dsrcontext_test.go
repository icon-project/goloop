/*
 * Copyright 2023 ICON Foundation
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
 *
 */

package state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
)

func TestNewDoubleSignContext_Basic(t *testing.T) {
	dbase := db.NewMapDB()
	vs0 := newDummyValidatorsFrom(10, 10)
	vss0, err := ValidatorSnapshotFromSlice(dbase, vs0)
	assert.NoError(t, err)
	ws := NewWorldState(dbase, nil, vss0, nil, nil)

	root, err := getDoubleSignContextRootOf(ws, 0)
	assert.NoError(t, err)
	assert.Nil(t, err)

	root, err = getDoubleSignContextRootOf(ws, module.AllRevision)
	assert.NoError(t, err)


	dsc1, err := root.ContextOf(module.DSTProposal)
	assert.NoError(t, err)

	_, err = root.ContextOf(module.DSTProposal+".INVALID")
	assert.Error(t, err)

	assert.EqualValues(t, root.Hash(), dsc1.Hash())

	bs := dsc1.Bytes()
	dsc2, err := decodeDoubleSignContext(module.DSTProposal, bs)
	assert.NoError(t, err)

	assert.EqualValues(t, dsc1.Hash(), dsc2.Hash())
	assert.EqualValues(t, dsc1.Bytes(), dsc2.Bytes())

	signer0 := vs0[0].Address()
	signer1 := dsc2.AddressOf(signer0.ID())
	assert.EqualValues(t, signer1, signer0)

	vu := newDummyValidator(1)
	signer2 := dsc2.AddressOf(vu.Address().ID())
	assert.Nil(t, signer2)

	assert.EqualValues(t, vss0.Hash(), dsc1.Hash(), dsc2.Hash())
}

func TestDecodeDoubleSignContext(t *testing.T) {
	cases := []struct {
		name  string
		dst   string
		data  []byte
		valid bool
	}{
		{"Valid", module.DSTVote, codec.BC.MustMarshalToBytes([][]byte{
			codec.BC.MustMarshalToBytes([]module.Address{newDummyAddress(10)}),
		}), true},
		{"Empty", module.DSTVote, codec.BC.MustMarshalToBytes([][]byte{}), false},
		{"Nil", module.DSTVote, nil, false},
		{"Bytes", module.DSTVote, codec.BC.MustMarshalToBytes([]byte{0x12, 0x23, 0x45}), false},
		{"Extra", module.DSTVote, append(codec.BC.MustMarshalToBytes([][]byte{
			codec.BC.MustMarshalToBytes([]module.Address{newDummyAddress(1)}),
		}), 0x12, 0x33), false},
		{"InvalidType", module.DSTVote+"x", codec.BC.MustMarshalToBytes([][]byte{
			codec.BC.MustMarshalToBytes([]module.Address{newDummyAddress(1)}),
		}), false},
		{"InvalidData", module.DSTVote, codec.BC.MustMarshalToBytes([]byte{0x12, 0x34}) , false},
	}
	for _, c := range cases {
		t.Run(fmt.Sprint(c.name), func(t *testing.T) {
			_, err := decodeDoubleSignContext(c.dst, c.data)
			if c.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}