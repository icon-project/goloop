package iiss

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func newDummyPRepInfo(i int) *icstate.PRepInfo {
	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)
	endpoint := fmt.Sprintf("%s.example.com:9080", name)
	return &icstate.PRepInfo{
		City:        &city,
		Country:     &country,
		Name:        &name,
		Email:       &email,
		WebSite:     &website,
		Details:     &details,
		P2PEndpoint: &endpoint,
	}
}

func newDummyExtensionState(t *testing.T) *ExtensionStateImpl {
	dbase := db.NewMapDB()
	ess := NewExtensionSnapshot(dbase, nil)
	es, ok := ess.NewState(false).(*ExtensionStateImpl)
	assert.True(t, ok)
	assert.NotNil(t, es)
	return es
}

type callContext struct {
	icmodule.CallContext
	from module.Address
	rev module.Revision
	blockHeight int64
}

func (cc *callContext) From() module.Address {
	return cc.from
}

func (cc *callContext) Revision() module.Revision {
	return cc.rev
}

func (cc *callContext) BlockHeight() int64 {
	return cc.blockHeight
}

func newDummyCallContext(from module.Address, rev module.Revision) icmodule.CallContext {
	return &callContext{
		from: from,
		rev: rev,
	}
}

func TestExtension_calculateRRep(t *testing.T) {
	type test struct {
		name           string
		totalSupply    *big.Int
		totalDelegated *big.Int
		rrep           *big.Int
	}

	tests := [...]test{
		{
			"MainNet-10,362,083-Decentralized",
			new(big.Int).Mul(new(big.Int).SetInt64(800326000), icutils.BigIntICX),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(170075049), icutils.BigIntICX),
				new(big.Int).SetInt64(583626807627704134),
			),
			new(big.Int).SetInt64(0x2ac),
		},
		{
			"MainNet-14,717,202",
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(819800188), icutils.BigIntICX),
				new(big.Int).SetInt64(205880949256032856),
			),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(203901940), icutils.BigIntICX),
				new(big.Int).SetInt64(576265206775030620),
			),
			new(big.Int).SetInt64(0x267),
		},
		{
			"MainNet-17,304,403",
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(831262951), icutils.BigIntICX),
				new(big.Int).SetInt64(723502790728839479),
			),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).SetInt64(234347234), icutils.BigIntICX),
				new(big.Int).SetInt64(465991733052158079),
			),
			new(big.Int).SetInt64(0x22c),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rrep := calculateRRep(tt.totalSupply, tt.totalDelegated)
			assert.Equal(t, 0, tt.rrep.Cmp(rrep), "%s\n%s", tt.rrep.String(), rrep.String())
		})
	}

	// from ICON1
	// index 56 : replace 239 to 240
	// 		there is some strange result in python
	//			>>> a: float = 56 / 100 * 10000
	//			>>> a
	//			5600.000000000001
	expectedRrepPerDelegatePercentage := [...]int64{
		1200,
		1171, 1143, 1116, 1088, 1062, 1035, 1010, 984, 959, 934,
		910, 886, 863, 840, 817, 795, 773, 751, 730, 710,
		690, 670, 650, 631, 613, 595, 577, 560, 543, 526,
		510, 494, 479, 464, 450, 435, 422, 408, 396, 383,
		371, 360, 348, 337, 327, 317, 307, 298, 290, 281,
		273, 266, 258, 252, 245, 240, 234, 229, 224, 220,
		216, 213, 210, 207, 205, 203, 201, 200, 200, 200,
		200, 200, 200, 200, 200, 200, 200, 200, 200, 200,
		200, 200, 200, 200, 200, 200, 200, 200, 200, 200,
		200, 200, 200, 200, 200, 200, 200, 200, 200, 200,
	}

	for i := 0; i < 101; i++ {
		name := fmt.Sprintf("delegated percentage: %d", i)
		t.Run(name, func(t *testing.T) {
			rrep := calculateRRep(new(big.Int).SetInt64(100), new(big.Int).SetInt64(int64(i)))
			assert.Equal(t, expectedRrepPerDelegatePercentage[i], rrep.Int64())
		})
	}
}

func TestExtension_validateRewardFund(t *testing.T) {
	type test struct {
		name           string
		iglobal        *big.Int
		totalSupply    *big.Int
		currentIglobal *big.Int
		err            bool
	}

	tests := [...]test{
		{
			"Inflation rate exceed 15%",
			new(big.Int).SetInt64(125),
			new(big.Int).SetInt64(1000),
			new(big.Int).SetInt64(101),
			true,
		},
		{
			"Inflation rate 18%",
			new(big.Int).SetInt64(150),
			new(big.Int).SetInt64(1000),
			new(big.Int).SetInt64(124),
			true,
		},
		{
			"Increase 10%",
			new(big.Int).SetInt64(110),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			false,
		},
		{
			"Decrease 10%",
			new(big.Int).SetInt64(90),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			false,
		},
		{
			"Increase 25%",
			new(big.Int).SetInt64(125),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			true,
		},
		{
			"Decrease 25%",
			new(big.Int).SetInt64(75),
			new(big.Int).SetInt64(120000),
			new(big.Int).SetInt64(100),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRewardFund(tt.iglobal, tt.currentIglobal, tt.totalSupply)
			if tt.err {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestExtensionStateImpl_getOldCommissionRate(t *testing.T) {
	var err error
	dbase := db.NewMapDB()
	ess := NewExtensionSnapshot(dbase, nil)
	es, ok := ess.NewState(false).(*ExtensionStateImpl)
	assert.True(t, ok)
	assert.NotNil(t, es)

	// Case
	// 0. No CommissionRate
	// 1. CommissionRate in Reward
	// 2. CommissionRate in Back2, Reward
	// 3. CommissionRate in Back1, Back2, Reward
	// 4. CommissionRate in Front, Back1, Back2, Reward

	var rate icmodule.Rate
	var expOldRate icmodule.Rate
	owner := common.MustNewAddressFromString("hx1234")

	for i := 0; i < 5; i++ {
		rate = icmodule.Rate(i)
		expOldRate = rate

		switch i {
		case 0: // None
		case 1: // Reward
			voted := icreward.NewVotedV2()
			voted.SetCommissionRate(rate)
			err = es.Reward.SetVoted(owner, voted)
			assert.NoError(t, err)
		case 2: // Back2
			err = es.Back2.AddCommissionRate(owner, rate)
			assert.NoError(t, err)
		case 3: // Back1
			err = es.Back1.AddCommissionRate(owner, rate)
			assert.NoError(t, err)
		case 4: // Front
			err = es.Front.AddCommissionRate(owner, rate)
			assert.NoError(t, err)
			expOldRate = icmodule.Rate(i-1)
		}

		rate, err = es.getOldCommissionRate(owner)
		if i == 0 {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, expOldRate, rate)
	}
}

func TestExtensionStateImpl_InitCommissionRate(t *testing.T) {

}

func TestExtensionStateImpl_SetCommissionRate(t *testing.T) {
	var err error
	owner := common.MustNewAddressFromString("hx1234")
	cc := newDummyCallContext(owner, icmodule.ValueToRevision(icmodule.RevisionPreIISS4))
	es := newDummyExtensionState(t)

	// Error: PRepBase Not Found
	err = es.SetCommissionRate(cc, icmodule.Rate(1))
	assert.Error(t, err)

	pi := newDummyPRepInfo(1)
	err = es.State.RegisterPRep(owner, pi, icmodule.BigIntZero, 0)
	assert.NoError(t, err)

	// Error: CommissionInfoNotFound
	err = es.SetCommissionRate(cc, icmodule.Rate(1))
	assert.Error(t, err)

	err = es.InitCommissionInfo(cc, icmodule.Rate(0), icmodule.ToRate(100), icmodule.Rate(5))
	assert.NoError(t, err)

	es.Back1 = es.Front
	es.Front = icstage.NewState(es.database)

	args := []struct{
		rate icmodule.Rate
		success bool
	}{
		{icmodule.Rate(-1), false},
		{icmodule.Rate(0), true},
		{icmodule.Rate(1), true},
		{icmodule.Rate(icmodule.DenomInRate), false},
		{icmodule.Rate(icmodule.DenomInRate+1), false},
	}

	var cr *icstage.CommissionRate
	var pb *icstate.PRepBaseState
	var rateInFront icmodule.Rate
	var oldRateInFront icmodule.Rate

	for _, arg := range args {
		pb = es.State.GetPRepBaseByOwner(owner, false)
		oldRate := pb.CommissionRate()
		cr, err = es.Front.GetCommissionRate(owner)
		assert.NoError(t, err)
		if cr != nil {
			oldRateInFront = cr.Value()
		}
		assert.Equal(t, oldRate, oldRateInFront)

		err = es.SetCommissionRate(cc, arg.rate)
		if arg.success {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}

		pb = es.State.GetPRepBaseByOwner(owner, false)
		cr, err = es.Front.GetCommissionRate(owner)
		if cr != nil {
			rateInFront = cr.Value()
		}
		if arg.success {
			assert.Equal(t, arg.rate, pb.CommissionRate())
			assert.Equal(t, arg.rate, rateInFront)
		} else {
			assert.Equal(t, oldRate, pb.CommissionRate())
			assert.Equal(t, oldRateInFront, rateInFront)
		}
	}
}
