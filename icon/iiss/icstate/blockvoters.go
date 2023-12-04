/*
 * Copyright 2021 ICON Foundation
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

package icstate

import (
	"fmt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

type blockVotersData struct {
	// voter is an owner address
	voterList []*common.Address
	voterMap  map[string]int
}

func (bvd *blockVotersData) init(voters []module.Address) {
	size := len(voters)
	bvd.voterList = make([]*common.Address, size)
	bvd.voterMap = make(map[string]int)

	for i, addr := range voters {
		bvd.voterList[i] = common.AddressToPtr(addr)
		bvd.voterMap[icutils.ToKey(addr)] = i
	}
}

func (bvd *blockVotersData) equal(other *blockVotersData) bool {
	if bvd == other {
		return true
	}
	if bvd.Len() != other.Len() {
		return false
	}
	for i, voter := range bvd.voterList {
		if !voter.Equal(other.voterList[i]) {
			return false
		}
	}
	return true
}

func (bvd *blockVotersData) IndexOf(owner module.Address) int {
	if i, ok := bvd.voterMap[icutils.ToKey(owner)]; !ok {
		return -1
	} else {
		return i
	}
}

func (bvd *blockVotersData) Get(i int) module.Address {
	if i < 0 || i >= bvd.Len() {
		return nil
	}
	return bvd.voterList[i]
}

func (bvd *blockVotersData) Len() int {
	return len(bvd.voterList)
}

func (bvd *blockVotersData) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		format := "blockVotersData"
		if f.Flag('+') {
			format += "{voterList=%v}"
		} else {
			format += "{%v}"
		}
		_, _ = fmt.Fprintf(f, format, bvd.voterList)
	case 's':
		_, _ = fmt.Fprintf(f, "voterList=%v", bvd.voterList)
	}
}

func newBlockVotersData(voters []module.Address) *blockVotersData {
	bv := new(blockVotersData)
	bv.init(voters)
	return bv
}

// =====================================================

type BlockVotersSnapshot struct {
	icobject.NoDatabase
	*blockVotersData
}

func (bvs *BlockVotersSnapshot) Version() int {
	return 0
}

func (bvs *BlockVotersSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	if err := decoder.DecodeAll(&bvs.voterList); err != nil {
		return err
	}

	bvs.voterMap = make(map[string]int)
	for i, voter := range bvs.voterList {
		bvs.voterMap[icutils.ToKey(voter)] = i
	}
	return nil
}

func (bvs *BlockVotersSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(bvs.voterList)
}

func (bvs *BlockVotersSnapshot) Equal(object icobject.Impl) bool {
	other, ok := object.(*BlockVotersSnapshot)
	if !ok {
		return false
	}
	if bvs == other {
		return true
	}
	if bvs == nil || other == nil {
		return false
	}
	return bvs.equal(other.blockVotersData)
}

func (bvs *BlockVotersSnapshot) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		format := "BlockVotersSnapshot"
		if f.Flag('+') {
			format += "{blockVotersData=%s}"
		} else {
			format += "{%s}"
		}
		_, _ = fmt.Fprintf(f, format, bvs.blockVotersData)
	case 's':
		_, _ = fmt.Fprintf(f, "%s", bvs.blockVotersData)
	}
}

func NewBlockVotersWithTag(_ icobject.Tag) *BlockVotersSnapshot {
	return &BlockVotersSnapshot{
		blockVotersData: newBlockVotersData(nil),
	}
}

func NewBlockVotersSnapshot(voters []module.Address) *BlockVotersSnapshot {
	return &BlockVotersSnapshot{
		blockVotersData: newBlockVotersData(voters),
	}
}
