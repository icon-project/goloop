package iiss

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func newDummyAddress(value int) module.Address {
	bs := make([]byte, common.AddressBytes)
	for i := 0; value != 0 && i < 8; i++ {
		bs[common.AddressBytes-1-i] = byte(value & 0xFF)
		value >>= 8
	}
	return common.MustNewAddress(bs)
}

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

type Call struct {
	method string
	params []interface{}
}

func (c *Call) Method() string {
	return c.method
}

func (c *Call) Params() []interface{} {
	return c.params
}

type CallTracer struct {
	callMap map[string][]*Call
}

func (ct *CallTracer) AddCall(method string, params ...interface{}) {
	ci := &Call{method, params}
	calls := ct.callMap[ci.method]
	ct.callMap[ci.method] = append(calls, ci)
}

func (ct *CallTracer) GetCalls(method string) []*Call {
	return ct.callMap[method]
}

func (ct *CallTracer) GetCall(method string, index int) *Call {
	if calls, ok := ct.callMap[method]; ok {
		return calls[index]
	}
	return nil
}

func (ct *CallTracer) Clear() {
	ct.callMap = make(map[string][]*Call)
}

func NewCallTracer() *CallTracer {
	return &CallTracer{
		callMap: make(map[string][]*Call),
	}
}

type mockCallContext struct {
	*CallTracer
	icmodule.CallContext
	from        module.Address
	rev         module.Revision
	blockHeight int64
}

func (cc *mockCallContext) From() module.Address {
	return cc.from
}

func (cc *mockCallContext) SetFrom(from module.Address) {
	cc.from = from
}

func (cc *mockCallContext) Revision() module.Revision {
	return cc.rev
}

func (cc *mockCallContext) SetRevision(rev module.Revision) {
	cc.rev = rev
}

func (cc *mockCallContext) BlockHeight() int64 {
	return cc.blockHeight
}

func (cc *mockCallContext) SetBlockHeight(blockHeight int64) {
	cc.blockHeight = blockHeight
}

func (cc *mockCallContext) OnEvent(addr module.Address, indexed, data [][]byte) {
	cc.AddCall("OnEvent", addr, indexed, data)
}

func (cc *mockCallContext) Withdraw(address module.Address, amount *big.Int, opType module.OpType) error {
	cc.AddCall("Withdraw", address, amount, opType)
	return nil
}

func (cc *mockCallContext) HandleBurn(address module.Address, amount *big.Int) error {
	cc.AddCall("HandleBurn", address, amount)
	return nil
}

func (cc *mockCallContext) Set(params map[string]interface{}) {
	for key, value := range params {
		switch key {
		case "rev", "revision":
			cc.rev = value.(module.Revision)
		case "bh", "blockHeight", "height":
			cc.blockHeight = value.(int64)
		case "from", "owner", "sender":
			cc.from = value.(module.Address)
		default:
			log.Panicf("UnexpectedName(%s)", key)
		}
	}
}

func (cc *mockCallContext) IncreaseBlockHeightBy(amount int64) int64 {
	cc.blockHeight += amount
	return cc.blockHeight
}

func newMockCallContext(params map[string]interface{}) *mockCallContext {
	cc := &mockCallContext{
		CallTracer: NewCallTracer(),
	}
	cc.Set(params)
	return cc
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
			voted.SetStatus(icmodule.ESEnable)
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
			expOldRate = icmodule.Rate(i - 1)
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
	rate := icmodule.ToRate(5)
	maxRate := icmodule.ToRate(30)
	maxChangeRate := icmodule.ToRate(10)

	var err error
	owner := common.MustNewAddressFromString("hx1234")
	cc := newMockCallContext(map[string]interface{}{
		"from": owner,
		"rev":  icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
	})
	es := newDummyExtensionState(t)

	pi := newDummyPRepInfo(1)
	err = es.State.RegisterPRep(owner, pi, icmodule.BigIntZero, 0)
	assert.NoError(t, err)

	err = es.InitCommissionInfo(cc, rate, maxRate, maxChangeRate)
	assert.NoError(t, err)

	pb := es.State.GetPRepBaseByOwner(owner, false)

	assert.Equal(t, rate, pb.CommissionRate())
	assert.Equal(t, maxRate, pb.MaxCommissionRate())
	assert.Equal(t, maxChangeRate, pb.MaxCommissionChangeRate())

	cr, err := es.Front.GetCommissionRate(owner)
	assert.NoError(t, err)
	assert.Equal(t, rate, cr.Value())

	// It is allowed only once to call InitCommissionInfo()
	err = es.InitCommissionInfo(
		cc, icmodule.ToRate(10), icmodule.ToRate(60), icmodule.ToRate(20))
	assert.Error(t, err)

	// Existing commissionInfo is not changed
	assert.Equal(t, rate, pb.CommissionRate())
	assert.Equal(t, maxRate, pb.MaxCommissionRate())
	assert.Equal(t, maxChangeRate, pb.MaxCommissionChangeRate())
}

func TestExtensionStateImpl_SetCommissionRate(t *testing.T) {
	var err error
	owner := common.MustNewAddressFromString("hx1234")
	cc := newMockCallContext(map[string]interface{}{
		"from": owner,
		"rev":  icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
	})
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

	args := []struct {
		rate    icmodule.Rate
		success bool
	}{
		{icmodule.Rate(-1), false},
		{icmodule.Rate(0), true},
		{icmodule.Rate(1), true},
		{icmodule.Rate(icmodule.DenomInRate), false},
		{icmodule.Rate(icmodule.DenomInRate + 1), false},
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

func TestExtensionStateImpl_SetSlashingRates(t *testing.T) {
	owner := common.MustNewAddressFromString("hx1234")
	cc := newMockCallContext(map[string]interface{}{
		"from": owner,
		"rev":  icmodule.ValueToRevision(icmodule.RevisionIISS4R0),
	})
	es := newDummyExtensionState(t)

	args := []struct {
		rates   map[string]icmodule.Rate
		success bool
	}{
		{
			map[string]icmodule.Rate{
				icmodule.PenaltyValidationFailure.String(): icmodule.ToRate(-5),
			},
			false,
		},
		{
			map[string]icmodule.Rate{
				icmodule.PenaltyValidationFailure.String(): icmodule.ToRate(1),
			},
			true,
		},
		{
			map[string]icmodule.Rate{
				icmodule.PenaltyValidationFailure.String():            icmodule.ToRate(0),
				icmodule.PenaltyAccumulatedValidationFailure.String(): icmodule.ToRate(20),
				icmodule.PenaltyPRepDisqualification.String():         icmodule.ToRate(100),
				icmodule.PenaltyDoubleSign.String():                   icmodule.ToRate(10),
				icmodule.PenaltyMissedNetworkProposalVote.String():    icmodule.ToRate(7),
			},
			true,
		},
		{
			map[string]icmodule.Rate{
				icmodule.PenaltyValidationFailure.String():            icmodule.ToRate(0),
				icmodule.PenaltyAccumulatedValidationFailure.String(): icmodule.ToRate(20),
				icmodule.PenaltyPRepDisqualification.String():         icmodule.ToRate(100),
				icmodule.PenaltyDoubleSign.String():                   icmodule.ToRate(-10),
				icmodule.PenaltyMissedNetworkProposalVote.String():    icmodule.ToRate(7),
			},
			false,
		},
	}

	jso, err := es.GetSlashingRates(cc, nil)
	assert.NoError(t, err)
	for key, value := range jso {
		assert.True(t, icmodule.ToPenaltyType(key) != icmodule.PenaltyNone)
		assert.True(t, icmodule.Rate(value.(int64)).IsValid())
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		rates := arg.rates

		t.Run(name, func(t *testing.T) {
			oldRates, err := es.GetSlashingRates(cc, nil)
			assert.NoError(t, err)

			err = es.SetSlashingRates(cc, rates)

			if !arg.success {
				assert.Error(t, err)
				jso, err = es.GetSlashingRates(cc, nil)
				assert.Equal(t, oldRates, jso)
				return
			}

			expRates := oldRates
			assert.NoError(t, err)
			for key, rate := range rates {
				expRates[key] = rate.NumInt64()
			}
			jso, err = es.GetSlashingRates(cc, nil)
			assert.NoError(t, err)
			assert.True(t, checkSlashingRates(jso))
			assert.Equal(t, expRates, jso)
		})
	}

	penaltyTypes := []icmodule.PenaltyType{
		icmodule.PenaltyDoubleSign,
		icmodule.PenaltyAccumulatedValidationFailure,
	}
	jso, err = es.GetSlashingRates(cc, penaltyTypes)
	assert.NoError(t, err)
	assert.Equal(t, len(penaltyTypes), len(jso))
	for _, pt := range penaltyTypes {
		_, ok := jso[pt.String()]
		assert.True(t, ok)
	}

	_, err = es.GetSlashingRates(cc, []icmodule.PenaltyType{icmodule.PenaltyNone})
	assert.Error(t, err)
}

func checkSlashingRates(rates map[string]interface{}) bool {
	for key, value := range rates {
		if pt := icmodule.ToPenaltyType(key); pt == icmodule.PenaltyNone {
			return false
		}
		rate := icmodule.Rate(value.(int64))
		if !rate.IsValid() {
			return false
		}
	}
	return true
}

func TestExtensionStateImpl_RequestUnjail(t *testing.T) {
	var err error
	rev := icmodule.RevisionIISS4R1
	owner := common.MustNewAddressFromString("hx1234")
	cc := newMockCallContext(map[string]interface{}{
		"from": owner,
		"rev":  icmodule.ValueToRevision(rev),
	})
	es := newDummyExtensionState(t)

	err = es.GenesisTerm(1000, rev)
	assert.NoError(t, err)

	pi := newDummyPRepInfo(1)
	err = es.RegisterPRep(cc, pi)
	assert.NoError(t, err)

	// Case of trying to request unjail for a normal PRep
	err = es.RequestUnjail(cc)
	assert.Error(t, err)

	// Non-existent PRep owner
	cc.SetFrom(common.MustNewAddressFromString("hx777"))
	err = es.RequestUnjail(cc)
	assert.Error(t, err)
}

func TestExtensionStateImpl_GetPRepStats(t *testing.T) {
	var err error
	size := 2
	rev := icmodule.RevisionIISS4R1
	bh := int64(1000)
	cc := newMockCallContext(map[string]interface{}{
		"rev":         icmodule.ValueToRevision(rev),
		"blockHeight": bh,
	})
	es := newDummyExtensionState(t)

	err = es.GenesisTerm(1000, rev)
	assert.NoError(t, err)

	for i := 0; i < size; i++ {
		cc.SetFrom(newDummyAddress(i + 1))
		pi := newDummyPRepInfo(i + 1)
		err = es.RegisterPRep(cc, pi)
		assert.NoError(t, err)
	}

	// Test GetPRepStats()
	cc.IncreaseBlockHeightBy(1)
	cc.SetFrom(common.MustNewAddressFromString("hx1234"))

	jso, err := es.GetPRepStats(cc)
	assert.NoError(t, err)
	assert.Equal(t, cc.BlockHeight(), jso["blockHeight"])

	preps := jso["preps"].([]interface{})
	assert.Equal(t, size, len(preps))

	exp := map[string]interface{}{
		"fail":         int64(0),
		"failCont":     int64(0),
		"grade":        int(icstate.GradeCandidate),
		"lastHeight":   int64(0),
		"penalties":    0,
		"realFail":     int64(0),
		"realFailCont": int64(0),
		"realTotal":    int64(0),
		"status":       int(icstate.Active),
		"total":        int64(0),
		"lastState":    int(icstate.None),
	}

	for i, prepInJSON := range preps {
		exp["address"] = newDummyAddress(size - i).String()
		jso = prepInJSON.(map[string]interface{})
		jso["address"] = jso["address"].(module.Address).String()
		assert.Equal(t, exp, jso)
	}

	// Test GetPRepStatsOf()
	cc.IncreaseBlockHeightBy(1)
	jso, err = es.GetPRepStatsOf(cc, common.MustNewAddressFromString("hx777"))
	assert.Nil(t, jso)
	assert.Error(t, err)

	address := newDummyAddress(1)
	jso, err = es.GetPRepStatsOf(cc, address)
	assert.NoError(t, err)
	assert.Equal(t, cc.BlockHeight(), jso["blockHeight"])

	preps = jso["preps"].([]interface{})
	assert.Equal(t, 1, len(preps))

	exp["address"] = address.String()
	prepInJSON := preps[0].(map[string]interface{})
	prepInJSON["address"] = prepInJSON["address"].(module.Address).String()
	assert.Equal(t, exp, prepInJSON)
}

func TestExtensionStateImpl_SetPRepCountConfig(t *testing.T) {
	var err error
	var main, sub, extra int64
	rev := icmodule.RevisionIISS4R1
	bh := int64(1000)
	cc := newMockCallContext(map[string]interface{}{
		"rev":         icmodule.ValueToRevision(rev),
		"blockHeight": bh,
	})
	es := newDummyExtensionState(t)

	args := []struct {
		counts  map[string]int64
		success bool
	}{
		{map[string]int64{"main": 22, "sub": 78, "extra": 3}, true},
		{map[string]int64{"main": 19, "sub": 81, "extra": 9}, true},
		{map[string]int64{"main": 25, "sub": 75}, true},
		{map[string]int64{"main": 10}, true},
		{map[string]int64{"sub": 40}, true},
		{map[string]int64{"extra": 3}, true},
		{map[string]int64{"main": 1001}, false},
		{map[string]int64{"sub": 10001}, false},
		{map[string]int64{"extra": 1001}, false},
		{map[string]int64{"main": -1}, false},
		{map[string]int64{"sub": -1}, false},
		{map[string]int64{"extra": -1}, false},
		{map[string]int64{"main": 0}, false},
		{map[string]int64{"main": 4, "sub": 2}, false},
		{map[string]int64{"main2": 4, "sub": 2}, false},
		{map[string]int64{"main": 4, "sub": 6, "extra": 0}, true},
		{map[string]int64{"extra": 1}, true},
	}

	for i, arg := range args {
		name := fmt.Sprintf("setPRepCountConfig-%02d", i)
		counts := arg.counts
		success := arg.success

		t.Run(name, func(t *testing.T) {
			err = es.SetPRepCountConfig(cc, counts)
			if success {
				assert.NoError(t, err)
				for k, v := range counts {
					switch k {
					case "main":
						main = v
					case "sub":
						sub = v
					case "extra":
						extra = v
					}
				}
			} else {
				assert.Error(t, err)
			}

			// Get all PRepCounts
			jso, err := es.GetPRepCountConfig(nil)
			assert.NoError(t, err)
			assert.Equal(t, main, jso["main"])
			assert.Equal(t, sub, jso["sub"])
			assert.Equal(t, extra, jso["extra"])
		})
	}

	args2 := []struct {
		names   []string
		success bool
	}{
		{nil, true},
		{[]string{}, true},
		{[]string{"main", "sub", "extra"}, true},
		{[]string{"main"}, true},
		{[]string{"sub"}, true},
		{[]string{"extra"}, true},
		{[]string{"main", "sub"}, true},
		{[]string{"main", "extra"}, true},
		{[]string{"sub", "extra"}, true},
		{[]string{"sub2", "extra"}, false},
		{[]string{"main", "main", "extra"}, false},
		{[]string{"main2"}, false},
		{[]string{"sub2"}, false},
		{[]string{"extra2"}, false},
	}

	for i, arg := range args2 {
		name := fmt.Sprintf("getPRepCountConfig-%02d", i)
		t.Run(name, func(t *testing.T){
			size := len(arg.names)
			if size == 0 {
				size = 3
			}
			jso, err := es.GetPRepCountConfig(arg.names)
			if arg.success {
				assert.Equal(t, size, len(jso))
				assert.NoError(t, err)
				for k, v := range jso {
					switch k {
					case "main":
						assert.Equal(t, main, v)
					case "sub":
						assert.Equal(t, sub, v)
					case "extra":
						assert.Equal(t, extra, v)
					}
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}
