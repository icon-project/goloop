/*
 * Copyright 2023 ICON Foundation
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

package icmodule

import (
	"fmt"
	"math/big"
	"strings"
)

const (
	denom = 10_000
)

type Rate int64

var bigIntDenom = big.NewInt(denom)

func (r Rate) Denom() int64 {
	return int64(denom)
}

func (r Rate) BigIntDenom() *big.Int {
	return bigIntDenom
}

func (r Rate) Num() int64 {
	return int64(r)
}

func (r Rate) BigIntNum() *big.Int {
	return big.NewInt(r.Num())
}

func (r Rate) MulInt64(v int64) int64 {
	return v * r.Num() / r.Denom()
}

func (r Rate) MulBigInt(v *big.Int) *big.Int {
	ret := new(big.Int).Set(v)
	ret = ret.Mul(ret, r.BigIntNum())
	return ret.Quo(ret, r.BigIntDenom())
}

func (r Rate) Percent() int64 {
	return r.Num() / 100
}

func (r Rate) String() string {
	q := r.Num() / r.Denom()
	rest := r.Num() % r.Denom()

	var sb strings.Builder
	if r < 0 {
		q *= -1
		rest *= -1  // abs(rest)
		sb.WriteByte('-')
	}
	sb.WriteString("%d.")

	if rest != 0 {
		size := 0
		for rest > 0 {
			if rest % 10 != 0 {
				break
			}
			size++
			rest /= 10
		}
		sb.WriteString(fmt.Sprintf("%%0%cd", '4' - rune(size)))
	} else {
		sb.WriteString("%d")
	}

	return fmt.Sprintf(sb.String(), q, rest)
}

func (r Rate) IsValid() bool {
	n := r.Num()
	return n >= 0 && n <= r.Denom()
}