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

package icreward

import (
	"math/big"

	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type IScore struct {
	icobject.ObjectBigInt
}

func (is *IScore) Equal(o icobject.Impl) bool {
	if is2, ok := o.(*IScore); ok {
		return is.Value.Cmp(is2.Value) == 0
	} else {
		return false
	}
}

func (is *IScore) Added(amount *big.Int) *IScore {
	n := new(IScore)
	if is == nil {
		n.Value = amount
	} else {
		n.Value = new(big.Int).Add(is.Value, amount)
	}
	return n
}

func newIScore(tag icobject.Tag) *IScore {
	return &IScore{
		*icobject.NewObjectBigInt(tag),
	}
}
