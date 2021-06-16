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
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icdb"
	"github.com/icon-project/goloop/icon/merkle/hexary"
)

type bpp struct {
	mt hexary.MerkleTree
}

func newBPP(dbase db.Database) (*bpp, error) {
	bpp := new(bpp)
	if err := bpp.init(dbase); err != nil {
		return nil, err
	}
	return bpp, nil
}

func (bpp *bpp) init(dbase db.Database) error {
	bk, err := dbase.GetBucket(icdb.BlockMerkle)
	if err != nil {
		return err
	}
	mt, err := hexary.NewMerkleTreeFromDB(bk, "", 0)
	if err != nil {
		return err
	}
	bpp.mt = mt
	return nil
}

func (bpp *bpp) GetBlockProof(height int64, opt int32) ([]byte, error) {
	if height >= bpp.mt.Cap() {
		return nil, nil
	}
	proof, err := bpp.mt.Prove(height, int(opt))
	if err != nil {
		return nil, err
	}
	bs := codec.MustMarshalToBytes(proof)
	return bs, nil
}
