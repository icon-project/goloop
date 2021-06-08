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

package icstate

import (
	"math/big"

	"github.com/icon-project/goloop/icon/iiss/icutils"
)

const (
	InitialIRep = 50_000 // in icx, not loop
	MinIRep     = 10_000
)

var BigIntZero = new(big.Int)
var BigIntInitialIRep = new(big.Int).Mul(new(big.Int).SetInt64(InitialIRep), icutils.BigIntICX)
var BigIntMinIRep = new(big.Int).Mul(new(big.Int).SetInt64(MinIRep), icutils.BigIntICX)