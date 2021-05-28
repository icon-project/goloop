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

package icstate

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestRewardCalcInfo(t *testing.T) {
	startHeight := int64(10)
	prevHeight := int64(5)

	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	rc1 := NewRewardCalcInfo()
	rc1.SetStartHeight(startHeight)
	rc1.SetPrevHeight(prevHeight)

	o1 := icobject.New(TypeRewardCalcInfo, rc1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	rc2 := ToRewardCalcInfo(o2)
	assert.True(t, rc1.Equal(rc2))
}

func TestRewardCalcInfo_Start(t *testing.T) {
	startHeight := int64(10)
	prevHeight := int64(5)

	rc1 := NewRewardCalcInfo()
	rc1.SetStartHeight(startHeight)
	rc1.SetPrevHeight(prevHeight)

	nBH := startHeight + int64(5)
	reward := int64(100)

	rc1.Update(nBH, new(big.Int).SetInt64(reward), nil)

	assert.Equal(t, nBH, rc1.StartHeight())
	assert.Equal(t, startHeight, rc1.PrevHeight())
	assert.Equal(t, startHeight-prevHeight, rc1.PrevPeriod())
	assert.Equal(t, reward, rc1.PrevCalcReward().Int64())
}
