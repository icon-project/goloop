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

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type LevelDB struct {
	leveldb *leveldb.DB
}

func (ds *LevelDB) GetBlockJSONByHeight(height int) ([]byte, error) {
	prefix := "block_height_key"
	key := make([]byte, len(prefix)+12)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[len(prefix)+4:], uint64(height))
	bid, err := ds.leveldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	blockjson, err := ds.leveldb.Get(bid, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return blockjson, nil
}

func (ds *LevelDB) GetLastBlockJSON() ([]byte, error) {
	bid, err := ds.leveldb.Get([]byte("last_block_key"), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	blockjson, err := ds.leveldb.Get(bid, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return blockjson, nil
}

func (ds *LevelDB) GetTransactionInfoJSONByTransaction(id []byte) ([]byte, error) {
	key := []byte(hex.EncodeToString(id))
	tinfo, err := ds.leveldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return tinfo, nil
}

func (ds *LevelDB) GetRepsJSONByHash(id []byte) ([]byte, error) {
	return nil, leveldb.ErrNotFound
}

func (ds *LevelDB) Close() error {
	return ds.leveldb.Close()
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
