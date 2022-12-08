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

package btp

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

func TestBTPBlockHeader_Basics(t *testing.T) {
	assert := assert.New(t)
	s := newComplexTestBuilderSetup(t)
	nts, err := s.bs.NetworkTypeSectionFor(1)
	assert.NoError(err)
	ns, err := nts.NetworkSectionFor(1)
	assert.NoError(err)
	bb, err := NewBTPBlockHeader(10, 0, nts, 1, module.FlagNextProofContext)
	assert.NoError(err)
	assert.EqualValues(10, bb.MainHeight())
	assert.EqualValues(0, bb.Round())
	assert.EqualValues(nts.NextProofContext().Hash(), bb.NextProofContextHash())
	assert.EqualValues(1, bb.NetworkID())
	assert.EqualValues(ns.UpdateNumber(), bb.UpdateNumber())
	assert.EqualValues(ns.FirstMessageSN(), bb.FirstMessageSN())
	assert.EqualValues(ns.NextProofContextChanged(), bb.NextProofContextChanged())
	assert.EqualValues(ns.PrevHash(), bb.PrevNetworkSectionHash())
	assert.EqualValues(ns.MessageCount(), bb.MessageCount())
	assert.EqualValues(ns.MessagesRoot(), bb.MessagesRoot())
	assert.EqualValues(nts.NextProofContext().Bytes(), bb.NextProofContext())
	nsToRoot, err := nts.NetworkSectionToRoot(1)
	assert.NoError(err)
	assert.EqualValues(nsToRoot, bb.NetworkSectionToRoot())
	bs := bb.HeaderBytes()
	var bb2 btpBlockHeader
	codec.MustUnmarshalFromBytes(bs, &bb2.format)
	assert.EqualValues(bb.(*btpBlockHeader).format, bb2.format)
}
