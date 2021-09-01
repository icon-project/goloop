/*
 * Copyright 2020 ICON Foundation
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

package service

import (
	"math/big"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/module"
)

type executionResult struct {
	patchReceipts  module.ReceiptList
	normalReceipts module.ReceiptList
	virtualFee     *big.Int
	totalFee       *big.Int
}

func (e *executionResult) PatchReceipts() module.ReceiptList {
	return e.patchReceipts
}

func (e *executionResult) NormalReceipts() module.ReceiptList {
	return e.normalReceipts
}

func (e *executionResult) VirtualFee() *big.Int {
	return e.virtualFee
}

func (e *executionResult) TotalFee() *big.Int {
	return e.totalFee
}

func NewExecutionResult(p, n module.ReceiptList, vfee, fee *big.Int) base.ExecutionResult {
	return &executionResult{p, n, vfee, fee}
}
