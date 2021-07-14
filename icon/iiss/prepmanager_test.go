package iiss

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
)

func createAddress(i int) module.Address {
	bs := make([]byte, common.AddressBytes)
	bs[20] = byte(i)
	address, err := common.NewAddress(bs)
	if err != nil {
		log.Fatalf("Invalid address: %#x", address)
	}

	return address
}

func newPRepInfo(i int) *icstate.PRepInfo {
	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)
	endpoint := fmt.Sprintf("%s.example.com:9080", name)

	return &icstate.PRepInfo{
		City: &city,
		Country: &country,
		Name: &name,
		Email: &email,
		WebSite: &website,
		Details: &details,
		P2PEndpoint: &endpoint,
	}
}

func newBond(address module.Address, amount int64) *icstate.Bond {
	return icstate.NewBond(common.AddressToPtr(address), big.NewInt(amount))
}

func newDelegation(address module.Address, amount int64) *icstate.Delegation {
	return icstate.NewDelegation(common.AddressToPtr(address), big.NewInt(amount))
}

func newDummyState(readonly bool) *icstate.State {
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	return icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), readonly, nil)
}

func createPRepManager(t *testing.T, readonly bool, size int) *PRepManager {
	state := newDummyState(readonly)

	for i := 0; i < size; i++ {
		owner := createAddress(i)
		ri := newPRepInfo(i)
		assert.NoError(t, state.RegisterPRep(owner, ri, icmodule.BigIntInitialIRep, 0))
	}

	return newPRepManager(state, nil)
}

//func compareRegInfo(prep *icstate.PRep, regInfo *icstate.RegInfo) bool {
//	return prep.City() == regInfo.city &&
//		prep.Country() == regInfo.country &&
//		prep.Details() == regInfo.details &&
//		prep.Email() == regInfo.email &&
//		prep.P2pEndpoint() == regInfo.p2pEndpoint &&
//		prep.Website() == regInfo.website &&
//		prep.Node() == regInfo.node &&
//		prep.Owner().Equal(regInfo.owner)
//}

func createBonds(start, size int) ([]*icstate.Bond, int64) {
	var sum int64
	ret := make([]*icstate.Bond, size, size)

	for i := 0; i < size; i++ {
		address := createAddress(start + i)
		amount := rand.Int63n(10000)
		ret[i] = newBond(address, amount)
		sum += amount
	}

	return ret, sum
}

func createDelegations(start, size int) ([]*icstate.Delegation, int64) {
	var sum int64
	ret := make([]*icstate.Delegation, size, size)

	for i := 0; i < size; i++ {
		address := createAddress(start + i)
		amount := rand.Int63n(10000)
		ret[i] = newDelegation(address, amount)
		sum += amount
	}

	return ret, sum
}

func TestPRepManager_ChangeDelegation(t *testing.T) {
	var err error
	size := 5

	pm := createPRepManager(t, false, size)
	state := pm.state

	dSize := 3
	ds0, sum0 := createDelegations(0, dSize)
	ds1, sum1 := createDelegations(0, dSize)
	ds2, sum2 := createDelegations(2, 3)
	exds := make(map[string]int64)

	type test struct {
		name    string
		ods     []*icstate.Delegation
		nds     []*icstate.Delegation
		sum     int64
		success bool
	}

	tests := []test{
		{
			name:    "(nil, ds0)",
			ods:     nil,
			nds:     ds0,
			sum:     sum0,
			success: true,
		},
		{
			name:    "(nil, ds1)",
			ods:     nil,
			nds:     ds1,
			sum:     sum0 + sum1,
			success: true,
		},
		{
			name:    "(ds0, ds1)",
			ods:     ds0,
			nds:     ds1,
			sum:     sum1 * 2,
			success: true,
		},
		{
			name:    "(ds1, ds0)",
			ods:     ds1,
			nds:     ds0,
			sum:     sum0 + sum1,
			success: true,
		},
		{
			name:    "(ds0, nil)",
			ods:     ds0,
			nds:     nil,
			sum:     sum1,
			success: true,
		},
		{
			name:    "(nil, ds2)",
			ods:     nil,
			nds:     ds2,
			sum:     sum1 + sum2,
			success: true,
		},
		{
			name:    "(ds1, nil)",
			ods:     ds1,
			nds:     nil,
			sum:     sum2,
			success: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ods := tt.ods
			nds := tt.nds
			sum := tt.sum
			success := tt.success

			for _, d := range nds {
				key := icutils.ToKey(d.To())
				exds[key] += d.Amount().Int64()
			}
			for _, d := range ods {
				key := icutils.ToKey(d.To())
				exds[key] -= d.Amount().Int64()
			}

			_, err = pm.ChangeDelegation(ods, nds)
			if success {
				assert.NoError(t, err)
				assert.Zero(t, state.GetTotalBond().Int64())
				assert.Equal(t, sum, state.GetTotalDelegation().Int64())

				for i := 0; i < size; i++ {
					owner := createAddress(i)
					prep := state.GetPRepByOwner(owner)
					assert.Equal(t, exds[icutils.ToKey(owner)], prep.Delegated().Int64())
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestPRepManager_ChangeBond(t *testing.T) {
	size := 5
	pm := createPRepManager(t, false, size)
	assert.Zero(t, pm.state.GetTotalDelegation().Int64())

	bs0, sum0 := createBonds(0, size)
	bs1, sum1 := createBonds(0, size)
	bs2, _ := createBonds(size, size)
	bs3, _ := createBonds(0, size)
	bs3[0].Value.SetValue(big.NewInt(-100))

	type test struct {
		name    string
		oBonds  []*icstate.Bond
		nBonds  []*icstate.Bond
		sum     int64
		success bool
	}

	tests := []test{
		{
			name:    "(nil,oBonds)",
			oBonds:  nil,
			nBonds:  bs0,
			sum:     sum0,
			success: true,
		},
		{
			name:    "(oBonds,nBonds)",
			oBonds:  bs0,
			nBonds:  bs1,
			sum:     sum1,
			success: true,
		},
		{
			name:    "(nBonds,nil)",
			oBonds:  bs1,
			nBonds:  nil,
			sum:     0,
			success: true,
		},
		{
			name:    "(nil,nBonds)-error",
			oBonds:  nil,
			nBonds:  bs2,
			sum:     0,
			success: false,
		},
		{
			name:    "(nil,nBonds)-error",
			oBonds:  nil,
			nBonds:  bs2,
			sum:     0,
			success: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := tt.oBonds
			nbs := tt.nBonds
			want := tt.sum
			success := tt.success

			_, err := pm.ChangeBond(obs, nbs)
			if success {
				assert.NoError(t, err)
				assert.Equal(t, want, pm.state.GetTotalBond().Int64())

				for j := 0; j < size; j++ {
					owner := createAddress(j)
					prep := pm.state.GetPRepByOwner(owner)
					bonded := prep.Bonded()
					assert.True(t, bonded.Int64() >= 0)

					if nbs == nil {
						assert.Zero(t, bonded.Int64())
					} else {
						assert.Zero(t, bonded.Cmp(nbs[j].Amount()))
					}
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestPRepManager_calculateRRep(t *testing.T) {
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
