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

package lcimporter

import (
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

type transitionID struct{ int }

type transition struct {
	parent *transition
	pid    *transitionID

	isSync       bool
	bi           module.BlockInfo
	transactions module.TransactionList

	worldSnapshot  trie.Immutable
	nextValidators module.ValidatorList
	receipts       module.ReceiptList
}

func (t *transition) PatchTransactions() module.TransactionList {
	return nil
}

func (t *transition) NormalTransactions() module.TransactionList {
	return t.transactions
}

func (t *transition) PatchReceipts() module.ReceiptList {
	return nil
}

func (t *transition) NormalReceipts() module.ReceiptList {
	return t.receipts
}

func (t *transition) Execute(cb module.TransitionCallback) (canceler func() bool, err error) {
	panic("implement me")
}

func (t *transition) ExecuteForTrace(ti module.TraceInfo) (canceler func() bool, err error) {
	panic("implement me")
}

func (t *transition) Result() []byte {
	return t.worldSnapshot.Hash()
}

func (t *transition) NextValidators() module.ValidatorList {
	return t.nextValidators
}

func (t *transition) LogsBloom() module.LogsBloom {
	return new(txresult.LogsBloom)
}

func (t *transition) BlockInfo() module.BlockInfo {
	return t.bi
}

func (t *transition) Equal(t2 module.Transition) bool {
	tr2, _ := t2.(*transition)
	if t == tr2 {
		return true
	}
	if t == nil || tr2 == nil {
		return false
	}
	if t.pid == tr2.pid {
		return true
	}
	// TODO
	return false
}

func CreateInitialTransition(dbase db.Database, result []byte, nvl module.ValidatorList) *transition {
	return &transition{
		pid: new(transitionID),

		isSync:       false,
		transactions: nil,

		worldSnapshot:  trie_manager.NewImmutable(dbase, result),
		nextValidators: nvl,
	}
}

func CreateSyncTransition(tr *transition) *transition {
	// TODO implement
	return nil
}
