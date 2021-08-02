package iiss

import (
	"math"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

// PRepManager manages PRepBase, PRepStatus and ActivePRep objects
type PRepManager struct {
	logger log.Logger
	state  *icstate.State
}

func (pm *PRepManager) ChangeDelegation(od, nd icstate.Delegations) (map[string]*big.Int, error) {
	delta := od.Delta(nd)
	delegatedToInactiveNode := big.NewInt(0)
	for key, value := range delta {
		owner, err := common.NewAddress([]byte(key))
		if err != nil {
			return nil, err
		}
		if value.Sign() != 0 {
			ps, _ := pm.state.GetPRepStatusByOwner(owner, true)
			ps.SetDelegated(new(big.Int).Add(ps.Delegated(), value))
			if !ps.IsActive() {
				delegatedToInactiveNode.Add(delegatedToInactiveNode, value)
			}
		}
	}

	oldTotalDelegation := pm.state.GetTotalDelegation()
	totalDelegation := new(big.Int).Set(oldTotalDelegation)
	totalDelegation.Add(totalDelegation, nd.GetDelegationAmount())
	totalDelegation.Sub(totalDelegation, od.GetDelegationAmount())
	//// Ignore the delegated amount to Inactive P-Rep
	totalDelegation.Sub(totalDelegation, delegatedToInactiveNode)

	if totalDelegation.Cmp(oldTotalDelegation) != 0 {
		if err := pm.state.SetTotalDelegation(totalDelegation); err != nil {
			return nil, err
		}
	}
	return delta, nil
}

func (pm *PRepManager) ChangeBond(oBonds, nBonds icstate.Bonds) (map[string]*big.Int, error) {
	delta := oBonds.Delta(nBonds)
	bondedToInactiveNode := big.NewInt(0)
	for key, value := range delta {
		owner, err := common.NewAddress([]byte(key))
		if err != nil {
			return nil, err
		}

		if value.Sign() != 0 {
			ps, _ := pm.state.GetPRepStatusByOwner(owner, false)
			if ps == nil {
				return nil, errors.Errorf("Failed to set bonded value to PRepStatus")
			}

			if ps.IsActive() {
				ps.SetBonded(new(big.Int).Add(ps.Bonded(), value))
			} else {
				// this code is not reachable, because there is no case of bonding to not-registered PRep
				bondedToInactiveNode.Add(bondedToInactiveNode, value)
			}
		}
	}

	oldTotalBond := pm.state.GetTotalBond()
	totalBond := new(big.Int).Set(oldTotalBond)
	totalBond.Add(totalBond, nBonds.GetBondAmount())
	totalBond.Sub(totalBond, oBonds.GetBondAmount())
	// Ignore the bonded amount to inactive P-Rep
	totalBond.Sub(totalBond, bondedToInactiveNode)

	if totalBond.Cmp(oldTotalBond) != 0 {
		if err := pm.state.SetTotalBond(totalBond); err != nil {
			return nil, err
		}
	}
	return delta, nil
}

func CalculateIRep(preps icstate.PRepSet, revision int) *big.Int {
	irep := new(big.Int)
	if revision < icmodule.RevisionDecentralize ||
		revision >= icmodule.RevisionICON2 {
		return irep
	}
	if revision >= icmodule.Revision9 {
		// set IRep via network proposal
		return nil
	}

	mainPRepCount := preps.GetPRepSize(icstate.GradeMain)
	totalDelegated := new(big.Int)
	totalWeightedIrep := new(big.Int)
	value := new(big.Int)

	for i := 0; i < mainPRepCount; i++ {
		prep := preps.GetPRepByIndex(i)
		totalWeightedIrep.Add(totalWeightedIrep, value.Mul(prep.IRep(), prep.Delegated()))
		totalDelegated.Add(totalDelegated, prep.Delegated())
	}

	if totalDelegated.Sign() == 0 {
		return irep
	}

	irep.Div(totalWeightedIrep, totalDelegated)
	if irep.Cmp(icmodule.BigIntMinIRep) == -1 {
		irep.Set(icmodule.BigIntMinIRep)
	}
	return irep
}

func CalculateRRep(totalSupply *big.Int, revision int, totalDelegation *big.Int) *big.Int {
	if revision < icmodule.RevisionIISS || revision >= icmodule.RevisionICON2 {
		// rrep is disabled
		return new(big.Int)
	}
	return calculateRRep(totalSupply, totalDelegation)
}

const (
	rrepMin        = 200   // 2%
	rrepMax        = 1_200 // 12%
	rrepPoint      = 7_000 // 70%
	rrepMultiplier = 10_000
)

func calculateRRep(totalSupply, totalDelegated *big.Int) *big.Int {
	ts := new(big.Float).SetInt(totalSupply)
	td := new(big.Float).SetInt(totalDelegated)
	delegatePercentage := new(big.Float).Quo(td, ts)
	delegatePercentage.Mul(delegatePercentage, new(big.Float).SetInt64(rrepMultiplier))
	dp, _ := delegatePercentage.Float64()
	if dp >= rrepPoint {
		return new(big.Int).SetInt64(rrepMin)
	}

	firstOperand := (rrepMax - rrepMin) / math.Pow(rrepPoint, 2)
	secondOperand := math.Pow(dp-rrepPoint, 2)
	return new(big.Int).SetInt64(int64(firstOperand*secondOperand + rrepMin))
}

func newPRepManager(state *icstate.State, logger log.Logger) *PRepManager {
	if logger == nil {
		logger = icutils.NewIconLogger(nil)
	}
	return &PRepManager{
		logger: logger,
		state:  state,
	}
}
