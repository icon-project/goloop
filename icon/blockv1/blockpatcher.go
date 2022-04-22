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

package blockv1

import (
	"bytes"
	_ "embed"
	"io"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
)

const (
	BH_41385450_RECORD = "\x67\x2f\x64\x68\xd1\x73\x04\x0f\x67\x7d\x4a\xa3\x8d\x7c\x33\xe1\x13\x44\x45\xc6\xa6\x7f\xfe\x85\x74\xff\xe2\xc3\xa4\x6c\x02\x69"
	BH_41385450_MISSED = "\x75\x8e\x2e\x93\xba\x3b\xe9\x3f\x45\x5c\x0e\x3f\x0b\x8b\x05\x09\x6d\x1a\xa6\x44\x8f\x14\x28\x1f\x10\xb8\xc0\x2a\x55\xda\x63\x95"
	BH_41385879_RECORD = "\x8c\x34\x09\xd4\x22\x73\xa5\x2f\x7f\xdf\xba\xbd\x2a\x59\x82\xc1\x90\x8a\xa4\x78\x03\x39\x4d\x8c\x8a\x7d\xd9\xb3\xd8\x89\x4a\x63"
	BH_41385879_MISSED = "\x55\x78\x3b\xad\xa1\x36\xea\x83\xb7\x7d\x4d\x08\x5d\xfc\xdb\x13\x09\x7f\x17\xe5\x64\x3d\x9c\x1e\xac\x1e\xac\xdf\xcb\x30\x6f\x7a"
)

var blockHashMap = map[string]string{
	BH_41385450_MISSED: BH_41385450_RECORD,
	BH_41385450_RECORD: BH_41385450_MISSED,
	BH_41385879_MISSED: BH_41385879_RECORD,
	BH_41385879_RECORD: BH_41385879_MISSED,
}

const (
	BH_41385450 = 41385450
	BH_41385879 = 41385879
)

//go:embed patch1.bin
var patchFor41385450 []byte

//go:embed patch2.bin
var patchFor41385879 []byte

func checkNeedPatch(dbase db.Database, record, missed string) (bool, error) {
	bk, err := dbase.GetBucket(db.BytesByHash)
	if err != nil {
		return false, err
	}
	value, err := bk.Get([]byte(record))
	if err != nil {
		return false, err
	}
	if len(value) == 0 {
		// Not in MainNet
		return false, nil
	}
	bhStored := string(crypto.SHA3Sum256(value))
	return bhStored != missed, nil
}

func applyPatch(dbase db.Database, databasePatch []byte) error {
	fd := bytes.NewReader(databasePatch)
	de := codec.BC.NewDecoder(fd)
	for {
		var bucketID db.BucketID
		var key []byte
		var value []byte
		err := de.DecodeListOf(&bucketID, &key, &value)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		bk, err := dbase.GetBucket(bucketID)
		if err != nil {
			return err
		}
		if len(value) > 0 {
			if err := bk.Set(key, value); err != nil {
				return err
			}
		} else {
			if err := bk.Delete(key); err != nil {
				return err
			}
		}
	}
}

func CheckAndApplyPatch(dbase db.Database) error {
	// check last height of the database
	lastHeight := block.GetLastHeightOf(dbase)
	if lastHeight >= BH_41385450 {
		if need, err := checkNeedPatch(dbase, BH_41385450_RECORD, BH_41385450_MISSED); err != nil {
			return err
		} else if need {
			if err = applyPatch(dbase, patchFor41385450); err != nil {
				return err
			}
		}
	}
	if lastHeight >= BH_41385879 {
		if need, err := checkNeedPatch(dbase, BH_41385879_RECORD, BH_41385879_MISSED); err != nil {
			return err
		} else if need {
			if err = applyPatch(dbase, patchFor41385879); err != nil {
				return err
			}
		}
	}
	return nil
}
