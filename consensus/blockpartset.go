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

package consensus

import (
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

type blockPartSet struct {
	PartSet

	// nil if PartSet is incomplete, or if we failed to convert complete PartSet
	// into module.BlockData
	block          module.BlockData
	validatedBlock module.BlockCandidate
}

func (bps *blockPartSet) Zerofy() {
	bps.PartSet = nil
	bps.block = nil
	bps.setValidatedBlock(nil)
}

func (bps *blockPartSet) ID() *PartSetID {
	if bps.PartSet == nil {
		return nil
	}
	return bps.PartSet.ID()
}

func (bps *blockPartSet) IsZero() bool {
	return bps.PartSet == nil && bps.block == nil
}

func (bps *blockPartSet) IsComplete() bool {
	return bps.PartSet != nil && bps.PartSet.IsComplete()
}

func (bps *blockPartSet) HasBlockData() bool {
	return bps.block != nil
}

func (bps *blockPartSet) Assign(oth *blockPartSet) {
	bps.PartSet = oth.PartSet
	bps.block = oth.block
	if oth.HasValidatedBlock() {
		bps.setValidatedBlock(oth.validatedBlock.Dup())
	} else {
		bps.setValidatedBlock(nil)
	}
}

func (bps *blockPartSet) AddPart(p Part, bm module.BlockManager) (added bool, err error) {
	if err := bps.PartSet.AddPart(p); err != nil {
		return false, err
	}
	if bps.PartSet.IsComplete() {
		blk, err := bm.NewBlockDataFromReader(bps.PartSet.NewReader())
		if err != nil {
			return true, err
		}
		bps.block = blk
	}
	return true, nil
}

func (bps *blockPartSet) SetByPartSetAndBlock(ps PartSet, blk module.BlockData) {
	prevID := bps.ID()
	bps.PartSet = ps
	bps.block = blk
	if !prevID.Equal(ps.ID()) {
		bps.setValidatedBlock(nil)
	}
}

// SetByPartSetAndValidatedBlock sets content of bps. Transfers ownership of bc to bps.
func (bps *blockPartSet) SetByPartSetAndValidatedBlock(ps PartSet, bc module.BlockCandidate) {
	bps.PartSet = ps
	bps.block = bc
	bps.setValidatedBlock(bc)
}

func (bps *blockPartSet) SetByPartSetID(psid *PartSetID) {
	if bps.ID().Equal(psid) {
		return
	}
	bps.PartSet = NewPartSetFromID(psid)
	bps.block = nil
	bps.setValidatedBlock(nil)
}

// SetByValidatedBlock sets validatedBlock. Transfers ownership of bc to bps.
func (bps *blockPartSet) SetByValidatedBlock(bc module.BlockCandidate) {
	psb := NewPartSetBuffer(ConfigBlockPartSize)
	log.Must(bc.MarshalHeader(psb))
	log.Must(bc.MarshalBody(psb))
	ps := psb.PartSet()
	bps.PartSet = ps
	bps.block = bc
	bps.setValidatedBlock(bc)
}

func (bps *blockPartSet) setValidatedBlock(bc module.BlockCandidate) {
	if bps.validatedBlock == bc {
		return
	}
	if bps.validatedBlock != nil {
		bps.validatedBlock.Dispose()
	}
	bps.validatedBlock = bc
}

func (bps *blockPartSet) HasValidatedBlock() bool {
	return bps.validatedBlock != nil
}

// AddPartFromBytes adds block part from block part bytes.
// Returns Part if the part is successfully added or nil if the part is not
// added.
func (bps *blockPartSet) AddPartFromBytes(bpBytes []byte, bm module.BlockManager) (Part, error) {
	bp, err := NewPart(bpBytes)
	if err != nil {
		return nil, err
	}
	if bps.GetPart(bp.Index()) != nil {
		return nil, nil
	}
	added, err := bps.AddPart(bp, bm)
	if !added && err != nil {
		return nil, err
	}
	if added && err != nil {
		log.Warnf("fail to create block. %+v", err)
	}
	return bp, nil
}
