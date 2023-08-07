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

package iiss

import (
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/service/state"
)

func recordSlashingRateChangedV2Event(cc icmodule.CallContext, penaltyType icmodule.PenaltyType, rate icmodule.Rate) {
	cc.OnEvent(state.SystemAddress,
		[][]byte{[]byte("SlashingRateChangedV2(str,int)")},
		[][]byte{
			[]byte(penaltyType.String()),
			intconv.Int64ToBytes(rate.NumInt64()),
		},
	)
}