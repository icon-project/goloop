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

type GetBlockTxCallback func([]*BlockTransaction, error)
type Canceler func()

type Executor struct {
}

// ProposeTransactions propose transactions for blocks to be consensus
// after finalized.
func (e *Executor) ProposeTransactions() ([]*BlockTransaction, error) {
	return nil, nil
}

// GetTransactions get already processed transactions in the range.
func (e *Executor) GetTransactions(from, to int64, callback GetBlockTxCallback) (Canceler, error) {
	return nil, nil
}

// FinalizeTransactions finalize transactions by specific range.
func (e *Executor) FinalizeTransactions(to int64) error {
	return nil
}

// SyncTransactions sync transactions
func (e *Executor) SyncTransactions([]*BlockTransaction) error {
	return nil
}
