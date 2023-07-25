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
	"math/big"
	"strconv"
	"strings"
)

const (
	DenomInRate  = int64(10_000)
	decimalPlace = 4
)

type Rate int64

var bigIntDenom = big.NewInt(DenomInRate)

// DenomInt64 returns denominator in int64
func (r Rate) DenomInt64() int64 {
	return DenomInRate
}

// DenomBigInt returns denominator in *BigInt
func (r Rate) DenomBigInt() *big.Int {
	return bigIntDenom
}

// NumInt64 returns numerator in int64
func (r Rate) NumInt64() int64 {
	return int64(r)
}

// NumBigInt returns numerator in *big.Int
func (r Rate) NumBigInt() *big.Int {
	return big.NewInt(r.NumInt64())
}

func (r Rate) MulInt64(v int64) int64 {
	ret := v * r.NumInt64()
	return ret / r.DenomInt64()
}

func (r Rate) MulBigInt(v *big.Int) *big.Int {
	ret := new(big.Int).Set(v)
	ret = ret.Mul(ret, r.NumBigInt())
	return ret.Quo(ret, r.DenomBigInt())
}

func (r Rate) Percent() int64 {
	return r.NumInt64() * 100 / r.DenomInt64()
}

func (r Rate) String() string {
	q := r.NumInt64() / r.DenomInt64()
	rest := r.NumInt64() % r.DenomInt64()

	var sb strings.Builder
	if r < 0 {
		q *= -1
		rest *= -1 // abs(rest)
		sb.WriteByte('-')
	}
	sb.WriteString(strconv.FormatInt(q, 10))
	sb.WriteByte('.')

	if rest != 0 {
		digits := make([]byte, 0, decimalPlace)
		for i := 0; i < decimalPlace; i++ {
			digit := rest % 10
			if digit != 0 || len(digits) > 0 {
				digits = append(digits, byte(digit))
			}
			rest /= 10
		}
		for i := len(digits) - 1; i >= 0; i-- {
			sb.WriteByte(byte('0') + digits[i])
		}
	} else {
		sb.WriteByte('0')
	}
	return sb.String()
}

func (r Rate) IsValid() bool {
	n := r.NumInt64()
	return n >= 0 && n <= r.DenomInt64()
}

func ToRate(percent int64) Rate {
	return Rate(percent * DenomInRate / 100)
}
