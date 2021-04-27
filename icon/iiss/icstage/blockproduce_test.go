/*
 * Copyright 2020 ICON Foundation
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

package icstage

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func Test_BlockVote(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)

	type_ := TypeBlockProduce
	version := 0
	proposerIndex := 3
	voteCount := 4
	voteMask := big.NewInt(0b1111)

	g1 := NewBlockProduce(proposerIndex, voteCount, voteMask)

	o1 := icobject.New(type_, g1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())
	assert.Equal(t, type_, o2.Tag().Type())
	assert.Equal(t, version, o2.Tag().Version())

	g2 := ToBlockProduce(o2)
	assert.Equal(t, true, g1.Equal(g2))
	assert.Equal(t, proposerIndex, g2.ProposerIndex())
	assert.Equal(t, voteCount, g2.VoteCount())
	assert.Equal(t, 0, voteMask.Cmp(g2.VoteMask()))
}
