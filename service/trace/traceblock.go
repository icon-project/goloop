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

package trace

import "github.com/icon-project/goloop/module"

type traceBlock struct {
	id []byte
	rl module.ReceiptList
}

func (b *traceBlock) ID() []byte {
	return b.id
}

func (b *traceBlock) GetReceipt(txIndex int) module.Receipt {
	if b.rl == nil {
		return nil
	}
	rct, _ := b.rl.Get(txIndex)
	return rct
}

func NewTraceBlock(id []byte, rl module.ReceiptList) module.TraceBlock {
	return &traceBlock{id, rl}
}
