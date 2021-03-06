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
	"encoding/json"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/blockv0"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

type Database interface {
	GetBlockJSONByHeight(height int) ([]byte, error)
	GetBlockJSONByID(id []byte) ([]byte, error)
	GetLastBlockJSON() ([]byte, error)
	GetResultJSON(id []byte) ([]byte, error)
	GetTransactionJSON(id []byte) ([]byte, error)
	GetRepsJSONByHash(id []byte) ([]byte, error)
	GetReceiptJSON(id []byte) ([]byte, error)
	Close() error
}

type Store struct {
	Database
	ReceiptRevision module.Revision
	ReceiptDatabase db.Database
}

func (lc *Store) GetBlockByHeight(height int) (blockv0.Block, error) {
	if bs, err := lc.Database.GetBlockJSONByHeight(height); err != nil {
		return nil, err
	} else {
		b, err := blockv0.ParseBlock(bs, lc)
		if err != nil {
			log.Warnf("Fail to parse block err=%+v blocks=%s", err, string(bs))
		}
		return b, err
	}
}

func (lc *Store) GetLastBlock() (blockv0.Block, error) {
	if bs, err := lc.Database.GetLastBlockJSON(); err != nil {
		return nil, err
	} else {
		b, err := blockv0.ParseBlock(bs, lc)
		if err != nil {
			log.Warnf("Fail to parse block err=%+v blocks=%s", err, string(bs))
		}
		return b, err
	}
}

type TransactionInfo struct {
	BlockID     common.HexBytes     `json:"block_hash"`
	BlockHeight int                 `json:"block_height"`
	TxIndex     common.HexInt32     `json:"tx_index"`
	Transaction blockv0.Transaction `json:"transaction"`
	Receipt     json.RawMessage     `json:"receipt"`
	Result      json.RawMessage     `json:"result"`
}

func (lc *Store) GetResult(id []byte) (*TransactionInfo, error) {
	bs, err := lc.GetResultJSON(id)
	if err != nil {
		return nil, err
	}
	tinfo := new(TransactionInfo)
	if err := json.Unmarshal(bs, tinfo); err != nil {
		return nil, err
	}
	return tinfo, nil
}

func (lc *Store) GetTransaction(id []byte) (*blockv0.Transaction, error) {
	bs, err := lc.Database.GetTransactionJSON(id)
	if err != nil {
		return nil, err
	}
	var tx blockv0.Transaction
	if err := json.Unmarshal(bs, &tx); err != nil {
		return nil, err
	} else {
		return &tx, nil
	}
}

func (lc *Store) GetReceipt(id []byte) (module.Receipt, error) {
	if rct, err := lc.GetReceiptJSON(id); err != nil {
		return nil, errors.Wrap(err, "FailureInGetResultJSON")
	} else {
		if r, err := txresult.NewReceiptFromJSON(lc.ReceiptDatabase, lc.ReceiptRevision, rct); err != nil {
			log.Warnf("FailureInParsingJSON(json=%q)", string(rct))
			return nil, err
		} else {
			return r, nil
		}
	}
}

func (lc *Store) GetRepsByHash(id []byte) (*blockv0.RepsList, error) {
	js, err := lc.GetRepsJSONByHash(id)
	if err != nil {
		return nil, err
	}
	reps := new(blockv0.RepsList)
	if err := json.Unmarshal(js, reps); err != nil {
		return nil, err
	}
	return reps, nil
}

func (lc *Store) SetReceiptParameter(dbase db.Database, rev module.Revision) {
	lc.ReceiptDatabase = dbase
	lc.ReceiptRevision = rev
}

func OpenStore(blockuri string) (*Store, error) {
	lcdb := new(Store)
	if strings.HasPrefix(blockuri, "http://") ||
		strings.HasPrefix(blockuri, "https://") {
		if bs, err := OpenNodeDB(blockuri); err != nil {
			return nil, err
		} else {
			lcdb.Database = bs
		}
	} else {
		if bs, err := OpenLevelDB(blockuri); err != nil {
			return nil, err
		} else {
			lcdb.Database = bs
		}
	}
	return lcdb, nil
}
