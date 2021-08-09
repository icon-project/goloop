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
	"encoding/json"
	"fmt"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/transaction"

	"github.com/icon-project/goloop/module"
)

const (
	Version01a = "0.1a"
	Version03  = "0.3"
	Version04  = "0.4"
	Version05  = "0.5"
)

type Transaction struct {
	module.Transaction
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	jso, err := t.Transaction.ToJSON(module.JSONVersionLast)
	if err != nil {
		return nil, err
	} else {
		return json.Marshal(jso)
	}
}

func (t *Transaction) UnmarshalJSON(b []byte) error {
	if tr, err := transaction.NewTransactionFromJSON(b); err != nil {
		return err
	} else {
		t.Transaction = tr
		return nil
	}
}

func (t Transaction) String() string {
	return fmt.Sprint(t.Transaction)
}

type Block interface {
	Version() string
	ID() []byte
	Height() int64
	Timestamp() int64
	PrevID() []byte
	Votes() *BlockVoteList
	Proposer() module.Address
	Validators() *RepsList
	NextValidators() *RepsList
	NormalTransactions() []module.Transaction
	LogsBloom() module.LogsBloom
	Verify(prev Block) error
	ToJSON(version module.JSONVersion) (interface{}, error)
	TransactionRoot() []byte
}

type Store interface {
	GetRepsByHash(hash []byte) (*RepsList, error)
}

type blockJSON struct {
	Version string `json:"version"`
}

func ParseBlock(b []byte, lc Store) (Block, error) {
	rawBlk := new(blockJSON)
	if err := json.Unmarshal(b, rawBlk); err != nil {
		return nil, err
	}
	switch rawBlk.Version {
	case Version01a:
		return ParseBlockV01a(b)
	case Version03, Version04, Version05:
		return ParseBlockV03(b, lc)
	default:
		return nil, errors.UnsupportedError.Errorf(
			"UnknownBlockVersion(version=%s)", rawBlk.Version)
	}
}
