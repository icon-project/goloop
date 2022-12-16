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

import "github.com/icon-project/goloop/module"

type blockPartSet struct {
	PartSet

	// nil if partset is incomplete
	block          module.BlockData
	validatedBlock module.BlockCandidate
}

func (bps *blockPartSet) Zerofy() {
	bps.PartSet = nil
	bps.block = nil
	bps.SetValidatedBlock(nil)
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
	return bps.block != nil
}

func (bps *blockPartSet) Assign(oth *blockPartSet) {
	bps.PartSet = oth.PartSet
	bps.block = oth.block
	if oth.HasValidatedBlock() {
		bps.SetValidatedBlock(oth.validatedBlock.Dup())
	} else {
		bps.SetValidatedBlock(nil)
	}
}

// Set sets content of bps. Transfers ownership of bc to bps.
func (bps *blockPartSet) Set(ps PartSet, blk module.BlockData, bc module.BlockCandidate) {
	bps.PartSet = ps
	bps.block = blk
	bps.SetValidatedBlock(bc)
}

// SetValidatedBlock sets validatedBlock. Transfers ownership of bc to bps.
func (bps *blockPartSet) SetValidatedBlock(bc module.BlockCandidate) {
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
