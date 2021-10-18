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

package lcstore

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
)

type LevelDB struct {
	leveldb *leveldb.DB
}

func (ds *LevelDB) get(key []byte) ([]byte, error) {
	bs, err := ds.leveldb.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return nil, errors.ErrNotFound
	}
	return bs, err
}

func (ds *LevelDB) GetBlockJSONByHeight(height int, pre bool) ([]byte, error) {
	prefix := "block_height_key"
	key := make([]byte, len(prefix)+12)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix)+4:], uint64(height))
	bid, err := ds.get(key)
	if err != nil {
		if err == errors.ErrNotFound && pre {
			return ds.get([]byte("shutdown_unconfirmed_block"))
		}
		return nil, err
	}
	return ds.GetBlockJSONByID(bid)
}

func (ds *LevelDB) GetBlockJSONByID(bid []byte) ([]byte, error) {
	blockjson, err := ds.get(bid)
	if err != nil {
		return nil, err
	}
	return blockjson, nil
}

func (ds *LevelDB) GetLastBlockJSON() ([]byte, error) {
	bid, err := ds.get([]byte("last_block_key"))
	if err != nil {
		return nil, err
	}
	blockjson, err := ds.get(bid)
	if err != nil {
		return nil, err
	}
	return blockjson, nil
}

func (ds *LevelDB) GetResultJSON(id []byte) ([]byte, error) {
	key := []byte(hex.EncodeToString(id))
	tinfo, err := ds.get(key)
	if err != nil {
		return nil, err
	}
	return tinfo, nil
}

type rawTransactionInfo struct {
	BlockID     common.HexBytes `json:"block_hash"`
	BlockHeight int             `json:"block_height"`
	TxIndex     common.HexInt32 `json:"tx_index"`
	Transaction json.RawMessage `json:"transaction"`
	Result      json.RawMessage `json:"result"`
}

func (ds *LevelDB) GetTransactionJSON(id []byte) ([]byte, error) {
	ti, err := ds.GetResultJSON(id)
	if err != nil {
		return nil, err
	}
	var tinfo rawTransactionInfo
	if err := json.Unmarshal(ti, &tinfo); err != nil {
		return nil, err
	} else {
		return tinfo.Transaction, nil
	}
}

func (ds *LevelDB) GetReceiptJSON(id []byte) ([]byte, error) {
	ti, err := ds.GetResultJSON(id)
	if err != nil {
		return nil, err
	}
	var tinfo rawTransactionInfo
	if err := json.Unmarshal(ti, &tinfo); err != nil {
		return nil, err
	}
	return tinfo.Result, nil
}

func (ds *LevelDB) GetRepsJSONByHash(id []byte) ([]byte, error) {
	key := append([]byte("preps_key"), id...)
	res, err := ds.get(key)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (ds *LevelDB) Close() error {
	return ds.leveldb.Close()
}

func (ds *LevelDB) GetTPS() float32 {
	return 0
}

func OpenLevelDB(dir string) (Database, error) {
	opts := &opt.Options{
		ReadOnly: true,
	}
	if db, err := leveldb.OpenFile(dir, opts); err != nil {
		return nil, err
	} else {
		return &LevelDB{db}, nil
	}
}
