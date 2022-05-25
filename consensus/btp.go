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

const (
	FlagBTPBlockHeader      = 0x1
	FlagBTPBlockProof       = 0x2
	FlagIncludeProofContext = 0x3
)

// GetBTPBlockHeaderAndProof returns header and proof according to the given
// flag.
func (cs *consensus) GetBTPBlockHeaderAndProof(
	blk module.Block,
	nid int64,
	flag uint,
) (module.BTPBlockHeader, []byte, error) {
	return nil, nil, nil
}
