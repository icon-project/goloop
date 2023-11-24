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
 *
 */

package contract

import (
	"math/big"

	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

const (
	EventRevisionSet  = "RevisionSet(int)"
	EventStepPriceSet = "StepPriceSet(int)"
	EventStepCostSet  = "StepCostSet(str,int)"

	EventMaxStepLimitSet = "MaxStepLimitSet(str,int)"
	EventTimestampThresholdSet = "TimestampThresholdSet(int)"
)

func GetRevision(cc CallContext) int {
	as := cc.GetAccountState(state.SystemID)
	return int(scoredb.NewVarDB(as, state.VarRevision).Int64())
}

// SetRevision set revision value in system storage.
// then returns old value on success.
// If a new revision is smaller than the current, then it
// returns InvalidParameterError
// If the revision is changed, then it emits proper events
// Otherwise, it returns old revision value.
func SetRevision(cc CallContext, revision int, ignoreSame bool) (int, error) {
	as := cc.GetAccountState(state.SystemID)
	vdb := scoredb.NewVarDB(as, state.VarRevision)
	oldRevision := int(vdb.Int64())
	if revision <= oldRevision {
		if revision == oldRevision && ignoreSame {
			return oldRevision, nil
		}
		return oldRevision, scoreresult.InvalidParameterError.Errorf(
			"InvalidRevisionValue(old=%d,new=%d)", oldRevision, revision)
	}
	if err := vdb.Set(revision); err != nil {
		return oldRevision, err
	}
	if cc.Revision().Has(module.ReportConfigureEvents) {
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{[]byte(EventRevisionSet)},
			[][]byte{intconv.Int64ToBytes(int64(revision))},
		)
	}
	return oldRevision, nil
}

func GetStepPrice(cc CallContext) *big.Int {
	as := cc.GetAccountState(state.SystemID)
	return intconv.BigIntSafe(scoredb.NewVarDB(as, state.VarStepPrice).BigInt())
}

func SetStepPrice(cc CallContext, price *big.Int) (bool, error) {
	as := cc.GetAccountState(state.SystemID)
	vdb := scoredb.NewVarDB(as, state.VarStepPrice)
	if v := vdb.BigInt() ; v != nil && v.Cmp(price) == 0 {
		return false, nil
	}
	if price.Sign() < 0 {
		return false, scoreresult.InvalidParameterError.Errorf(
			"InvalidStepPrice(price=%s)", price)
	}
	if err := vdb.Set(price); err != nil {
		return false, err
	}
	if cc.Revision().Has(module.ReportConfigureEvents) {
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{ []byte(EventStepPriceSet) },
			[][]byte{ intconv.BigIntToBytes(price) },
		)
	}
	return true, nil
}

func GetStepCost(cc CallContext, name string) *big.Int {
	as := cc.GetAccountState(state.SystemID)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	return containerdb.BigIntSafe(stepCostDB.Get(name))
}

func GetStepCosts(cc CallContext) map[string]any {
	stepCosts := make(map[string]any)
	as := cc.GetAccountState(state.SystemID)
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)
	cnt := stepTypes.Size()
	for i := 0; i < cnt; i++ {
		name := stepTypes.Get(i).String()
		value := stepCostDB.Get(name).BigInt()
		stepCosts[name] = value
	}
	return stepCosts
}

func SetStepCost(cc CallContext, name string, cost *big.Int, prune bool) (bool, error) {
	if !state.IsValidStepType(name) {
		return false, scoreresult.InvalidParameterError.Errorf("InvalidStepType(name=%s)", name)
	}
	deleteCost := cost.Sign()==0 && prune

	as := cc.GetAccountState(state.SystemID)
	stepTypes := scoredb.NewArrayDB(as, state.VarStepTypes)
	stepCostDB := scoredb.NewDictDB(as, state.VarStepCosts, 1)

	value := stepCostDB.Get(name)
	old := containerdb.BigIntSafe(value)
	if deleteCost {
		if value == nil {
			return false, nil
		}
		size := stepTypes.Size()
		for i:=0 ; i<size ; i++ {
			if stepTypes.Get(i).String() == name {
				last := stepTypes.Pop()
				if i!=size-1 {
					if err := stepTypes.Set(i, last); err != nil {
						return false, err
					}
				}
				if err := stepCostDB.Delete(name); err != nil {
					return false, err
				}
				break
			}
		}
	} else {
		if value == nil {
			if err := stepTypes.Put(name); err != nil {
				return false, err
			}
		}
		if err := stepCostDB.Set(name, cost); err != nil {
			return false, err
		}
	}

	if old.Cmp(cost) == 0 {
		return false, nil
	}
	if cc.Revision().Has(module.ReportConfigureEvents) {
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{[]byte("StepCostSet(str,int)")},
			[][]byte{[]byte(name), intconv.BigIntToBytes(cost)},
		)
	}
	return true, nil
}

// GetMaxStepLimit returns MaxStepLimit for specified operation
func GetMaxStepLimit(cc CallContext, name string) *big.Int {
	as := cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	return containerdb.BigIntSafe(stepLimitDB.Get(name))
}

func SetMaxStepLimit(cc CallContext, name string, cost *big.Int) (bool, error) {
	if !state.IsValidStepLimitType(name) {
		return false, scoreresult.InvalidParameterError.Errorf("InvalidStepLimitType(name=%s)", name)
	}
	if cost.Sign() < 0 {
		return false, scoreresult.InvalidParameterError.Errorf(
			"InvalidMaxStepLimit(negative v=%v)", cost)
	}
	costIsZero := cost.Sign()==0

	as := cc.GetAccountState(state.SystemID)
	stepLimitDB := scoredb.NewDictDB(as, state.VarStepLimit, 1)
	stepLimitTypes := scoredb.NewArrayDB(as, state.VarStepLimitTypes)
	if value := stepLimitDB.Get(name) ; value == nil {
		if costIsZero {
			return false, nil
		}
		if err := stepLimitTypes.Put(name); err != nil {
			return false, err
		}
	} else if value.BigInt().Cmp(cost) == 0 {
		return false, nil
	}
	if err := stepLimitDB.Set(name, cost); err != nil {
		return false, err
	}
	if cc.Revision().Has(module.ReportConfigureEvents) {
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{ []byte(EventMaxStepLimitSet) },
			[][]byte{ []byte(name), intconv.BigIntToBytes(cost) },
		)
	}
	return true, nil
}

func GetTimestampThreshold(cc CallContext) int64 {
	as := cc.GetAccountState(state.SystemID)
	db := scoredb.NewVarDB(as, state.VarTimestampThreshold)
	return db.Int64()
}

func SetTimestampThreshold(cc CallContext, value int64) (bool, error) {
	as := cc.GetAccountState(state.SystemID)
	if value < 0 {
		return false, scoreresult.InvalidParameterError.Errorf("Negative threshold value=%d", value)
	}
	db := scoredb.NewVarDB(as, state.VarTimestampThreshold)
	if old := db.Int64(); old == value {
		return false, nil
	}
	if value == 0 {
		if _, err := db.Delete(); err != nil {
			return false, err
		}
	} else {
		if err := db.Set(value); err != nil {
			return false, err
		}
	}
	if cc.Revision().Has(module.ReportConfigureEvents) {
		cc.OnEvent(
			state.SystemAddress,
			[][]byte{ []byte(EventTimestampThresholdSet) },
			[][]byte{ intconv.Int64ToBytes(value) },
		)
	}
	return true, nil
}
