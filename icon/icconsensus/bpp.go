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

package icconsensus

import (
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/merkle/hexary"
)

type bpp struct {
	hexary.MerkleTree
}

func newBPP(mt hexary.MerkleTree) *bpp {
	return &bpp{ mt }
}

func (bpp *bpp) GetBlockProof(height int64, opt int32) ([]byte, error) {
	if height >= bpp.Cap() {
		return nil, nil
	}
	proof, err := bpp.Prove(height, int(opt))
	if err != nil {
		return nil, err
	}
	bs := codec.MustMarshalToBytes(proof)
	return bs, nil
}
