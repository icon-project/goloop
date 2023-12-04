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

package blockv0

import (
	"encoding/hex"
	"encoding/json"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/merkle"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/transaction"
	"github.com/icon-project/goloop/service/txresult"
)

func CalcMerkleRootOfReceiptSlice(
	receipts []txresult.Receipt,
	txs []module.Transaction,
	height int64,
) []byte {
	items := make([]merkle.Item, len(receipts))
	for i, r := range receipts {
		items[i] = merkle.HashedItem(CalcOriginalReceiptHash(
			r,
			txs[i].ID(),
			height,
			i,
		))
	}
	return merkle.CalcHashOfList(items)
}

func CalcMerkleRootOfReceiptList(
	receipts module.ReceiptList,
	txs module.TransactionList,
	height int64,
) []byte {
	var items []merkle.Item
	txIter := txs.Iterator()
	for rIter := receipts.Iterator(); rIter.Has(); _, _ = rIter.Next(), txIter.Next() {
		r, _ := rIter.Get()
		tx, i, _ := txIter.Get()
		items = append(
			items,
			merkle.HashedItem(CalcOriginalReceiptHash(r, tx.ID(), height, i)),
		)
	}
	return merkle.CalcHashOfList(items)
}

var receiptsSalt = []byte("icx_receipt.")

func CalcOriginalReceiptHash(
	r module.Receipt,
	txHash []byte,
	height int64,
	txIndex int,
) []byte {
	m, err := r.ToJSON(module.JSONVersion3)
	log.Must(err)

	ma := m.(map[string]interface{})
	ma["txHash"] = hex.EncodeToString(txHash)
	ma["blockHeight"] = intconv.FormatInt(height)
	ma["txIndex"] = intconv.FormatInt(int64(txIndex))
	bs, err := json.Marshal(ma)
	log.Must(err)
	bs, err = transaction.SerializeJSON(bs, nil, map[string]bool {
		"failure": true,
	})
	log.Must(err)

	bs = append(receiptsSalt, bs...)
	return crypto.SHA3Sum256(bs)
}
