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

package icstage

import (
	"math/big"

	"github.com/icon-project/goloop/icon/iiss/icobject"
)

type IScoreClaim struct {
	icobject.ObjectBigInt
}

func (ic *IScoreClaim) Equal(o icobject.Impl) bool {
	if ic2, ok := o.(*IScoreClaim); ok {
		return ic.Value.Cmp(ic2.Value) == 0
	} else {
		return false
	}
}

func (ic *IScoreClaim) Added(amount *big.Int) *IScoreClaim {
	n := new(IScoreClaim)
	if ic == nil {
		n.Value = amount
	} else {
		n.Value = new(big.Int).Add(ic.Value, amount)
	}
	return n
}

func newIScoreClaim(tag icobject.Tag) *IScoreClaim {
	return &IScoreClaim{
		*icobject.NewObjectBigInt(tag),
	}
}
