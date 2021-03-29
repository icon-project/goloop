package iiss

import (
	"fmt"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
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

func newRegInfo(i int) *RegInfo {
	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)
	endpoint := fmt.Sprintf("%s.example.com:9080/api/v3", name)
	node := module.Address(nil)
	owner := createAddress(i)

	return NewRegInfo(city, country, details, email, name, endpoint, website, node, owner)
}

func newBond(address module.Address, amount int64) *icstate.Bond {
	b := icstate.NewBond()
	b.Address.Set(address)
	b.Value.SetInt64(amount)
	return b
}

func newDelegation(address module.Address, amount int64) *icstate.Delegation {
	d := icstate.NewDelegation()
	d.Address.Set(address)
	d.Value.SetInt64(amount)
	return d
}

func createPRepManager(t *testing.T, readonly bool, size int) *PRepManager {
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	s := icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), readonly)
	pm := newPRepManager(s, nil)

	for i := 0; i < size; i++ {
		assert.NoError(t, pm.RegisterPRep(newRegInfo(i), BigIntInitialIRep))
	}
	pm.Sort()
	assert.NoError(t, pm.state.Flush())
	assert.Equal(t, 0, pm.GetPRepSize(icstate.Main))
	assert.Equal(t, 0, pm.GetPRepSize(icstate.Sub))
	assert.Equal(t, size, pm.GetPRepSize(icstate.Candidate))
	assert.Equal(t, size, pm.Size())
	return pm
}

func compareRegInfo(prep *PRep, regInfo *RegInfo) bool {
	return prep.City() == regInfo.city &&
		prep.Country() == regInfo.country &&
		prep.Details() == regInfo.details &&
		prep.Email() == regInfo.email &&
		prep.P2pEndpoint() == regInfo.p2pEndpoint &&
		prep.Website() == regInfo.website &&
		prep.Node() == regInfo.node &&
		prep.Owner().Equal(regInfo.owner)
}

func checkOrderedByBondedDelegation(pm *PRepManager, br int64) bool {
	prev := big.NewInt(math.MaxInt64)
	size := pm.Size()
	for i := 0; i < size; i++ {
		prep := pm.GetPRepByIndex(i)
		bd := prep.GetBondedDelegation(br)

		if prev.Cmp(bd) < 0 {
			return false
		}
	}
	return true
}

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

func createActivePRep(s *icstate.State, addr module.Address, bonded, delegated int64) {
	s.GetPRepBase(addr, true)
	ps := s.GetPRepStatus(addr, true)
	ps.SetStatus(icstate.Active)
	ps.SetBonded(big.NewInt(bonded))
	ps.SetDelegated(big.NewInt(delegated))
	s.AddActivePRep(addr)
}

// test for GetBondedDelegation
func TestPRepManager_Sort(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	s := icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), false)
	pm := newPRepManager(s, nil)

	br := int64(5)
	size := 10
	for i := 0; i < size; i++ {
		addr := createAddress(i + 1)
		bonded := rand.Int63()
		delegated := rand.Int63()
		createActivePRep(s, addr, bonded, delegated)
	}

	assert.NoError(t, pm.state.SetBondRequirement(5))
	pm.init()
	pm.sort()

	checkOrderedByBondedDelegation(pm, br)
}

func TestPRepManager_new(t *testing.T) {
	pm := createPRepManager(t, false, 0)
	assert.Zero(t, pm.Size())
	assert.Equal(t, 0, pm.GetPRepSize(icstate.Main))
	assert.Equal(t, 0, pm.GetPRepSize(icstate.Sub))
	assert.Equal(t, 0, pm.GetPRepSize(icstate.Candidate))
	assert.Zero(t, len(pm.orderedPReps))
	assert.Zero(t, len(pm.prepMap))
	assert.Zero(t, pm.TotalDelegated().Int64())
}

func TestPRepManager_Add(t *testing.T) {
	br := int64(5) // 5%
	size := 10
	pm := createPRepManager(t, false, size)
	assert.Equal(t, size, pm.Size())

	totalDelegated := big.NewInt(0)
	prev := big.NewInt(math.MaxInt64)
	for i := 0; i < size; i++ {
		prep := pm.GetPRepByIndex(i)
		bondedDelegation := prep.GetBondedDelegation(br)

		if prev.Cmp(bondedDelegation) < 0 {
			t.Errorf("PRepManager.Sort() is failed")
		}

		prev.Set(bondedDelegation)
		totalDelegated.Add(totalDelegated, prep.Delegated())
	}

	assert.NoError(t, pm.state.Flush())
	assert.Zero(t, totalDelegated.Cmp(pm.TotalDelegated()))
}

func TestPRepManager_RegisterPRep(t *testing.T) {
	size := 10
	pm := createPRepManager(t, false, 0)

	for i := 0; i < size; i++ {
		regInfo := newRegInfo(i)
		owner := regInfo.owner

		err := pm.RegisterPRep(regInfo, BigIntInitialIRep)
		assert.NoError(t, err)
		assert.Equal(t, i+1, pm.Size())

		owner = createAddress(i)
		prep := pm.GetPRepByOwner(owner)
		assert.Equal(t, icstate.Candidate, prep.Grade())
		assert.Equal(t, icstate.Active, prep.Status())
		assert.Equal(t, 0, BigIntInitialIRep.Cmp(prep.IRep()))
		assert.True(t, compareRegInfo(prep, regInfo))

		pb := pm.state.GetPRepBase(owner, false)
		assert.True(t, pb == prep.PRepBase)
		ps := pm.state.GetPRepStatus(owner, false)
		assert.True(t, ps == prep.PRepStatus)
	}

	assert.True(t, checkOrderedByBondedDelegation(pm, 5))
}

func TestPRepManager_RegisterPRepWithNotReadyPRep(t *testing.T) {
	size := 5
	start := 10
	br := int64(5)
	pm := createPRepManager(t, false, 0)

	nd, sum := createDelegations(start, size)
	delta, err := pm.ChangeDelegation(nil, nd)
	assert.NoError(t, err)
	etd := big.NewInt(sum)
	td := new(big.Int)
	for _, value := range delta {
		assert.True(t, value.Sign() >= 0)
		td.Add(td, value)
	}
	assert.Zero(t, etd.Cmp(td))

	for i := 0; i < size; i++ {
		regInfo := newRegInfo(start + i)
		err = pm.RegisterPRep(regInfo, BigIntInitialIRep)
		assert.NoError(t, err)
	}

	assert.Zero(t, etd.Cmp(pm.TotalDelegated()))
	assert.Equal(t, int64(0), pm.TotalBonded().Int64())
	checkOrderedByBondedDelegation(pm, br)
}

func TestPRepManager_disablePRep(t *testing.T) {
	size := 5
	pm := createPRepManager(t, false, size)
	assert.Equal(t, size, pm.Size())

	ss := []icstate.Status{
		icstate.Unregistered,
		icstate.Disqualified,
		icstate.Unregistered,
		icstate.Disqualified,
		icstate.Unregistered,
	}
	totalDelegated := new(big.Int).Set(pm.TotalDelegated())
	for i := 0; i < size; i++ {
		status := ss[i]
		owner := createAddress(i)
		prep := pm.GetPRepByOwner(owner)
		assert.True(t, prep.Owner().Equal(owner))

		err := pm.disablePRep(owner, status)
		assert.NoError(t, err)

		noPRep := pm.GetPRepByOwner(owner)
		assert.NotNil(t, noPRep)

		assert.Equal(t, size-i-1, pm.Size())

		totalDelegated.Sub(totalDelegated, prep.Delegated())
		assert.Zero(t, totalDelegated.Cmp(pm.TotalDelegated()))
		assert.Equal(t, status, prep.Status())
		assert.Equal(t, icstate.Candidate, prep.Grade())

		ps := pm.state.GetPRepStatus(owner, false)
		assert.True(t, ps == prep.PRepStatus)

		assert.Equal(t, size-i-1, pm.Size())
		assert.Zero(t, pm.GetPRepSize(icstate.Main))
		assert.Zero(t, pm.GetPRepSize(icstate.Sub))
		assert.Equal(t, size-i-1, pm.GetPRepSize(icstate.Candidate))
	}

	assert.Zero(t, pm.Size())
	assert.Zero(t, pm.GetPRepSize(icstate.Main))
	assert.Zero(t, pm.GetPRepSize(icstate.Sub))
	assert.Zero(t, pm.GetPRepSize(icstate.Candidate))
	assert.Zero(t, pm.TotalDelegated().Cmp(big.NewInt(0)))
}

func TestPRepManager_ChangeDelegation(t *testing.T) {
	var err error
	br := int64(5)
	size := 5

	pm := createPRepManager(t, false, size)
	assert.Zero(t, pm.TotalDelegated().Int64())

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
				assert.Zero(t, pm.TotalBonded().Int64())
				assert.Equal(t, sum, pm.TotalDelegated().Int64())
				assert.True(t, checkOrderedByBondedDelegation(pm, br))

				for i := 0; i < size; i++ {
					owner := createAddress(i)
					prep := pm.GetPRepByOwner(owner)
					assert.Equal(t, exds[icutils.ToKey(owner)], prep.Delegated().Int64())
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestPRepManager_ChangeBond(t *testing.T) {
	br := int64(5) // 5%
	size := 5
	pm := createPRepManager(t, false, size)
	assert.Zero(t, pm.TotalDelegated().Int64())

	bs0, sum0 := createBonds(0, size)
	bs1, sum1 := createBonds(0, size)
	bs2, _ := createBonds(size, size)
	bs3, _ := createBonds(0, size)
	bs3[0].Value.SetInt64(-100)

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
				assert.Equal(t, want, pm.TotalBonded().Int64())
				assert.True(t, checkOrderedByBondedDelegation(pm, br))

				for j := 0; j < size; j++ {
					owner := createAddress(j)
					prep := pm.GetPRepByOwner(owner)
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

func TestPRepManager_OnTermEnd(t *testing.T) {
	type test struct {
		size          int
		mainPRepCount int
		subPRepCount  int

		expectedMainPReps      int
		expectedSubPReps       int
		expectedCandidatePReps int
	}

	tests := [...]test{
		{
			size:                   10,
			mainPRepCount:          4,
			subPRepCount:           3,
			expectedMainPReps:      4,
			expectedSubPReps:       3,
			expectedCandidatePReps: 3,
		},
		{
			size:                   10,
			mainPRepCount:          8,
			subPRepCount:           12,
			expectedMainPReps:      8,
			expectedSubPReps:       2,
			expectedCandidatePReps: 0,
		},
		{
			size:                   10,
			mainPRepCount:          13,
			subPRepCount:           17,
			expectedMainPReps:      10,
			expectedSubPReps:       0,
			expectedCandidatePReps: 0,
		},
		{
			size:                   10,
			mainPRepCount:          13,
			subPRepCount:           17,
			expectedMainPReps:      10,
			expectedSubPReps:       0,
			expectedCandidatePReps: 0,
		},
	}

	for i, tt := range tests {
		bh := int64(123)
		name := fmt.Sprintf("test-%d", i)
		t.Run(name, func(t *testing.T) {
			pm := createPRepManager(t, false, tt.size)
			err := pm.OnTermEnd(tt.mainPRepCount, tt.subPRepCount, bh)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedMainPReps, pm.GetPRepSize(icstate.Main))
			assert.Equal(t, tt.expectedSubPReps, pm.GetPRepSize(icstate.Sub))
			assert.Equal(t, tt.expectedCandidatePReps, pm.GetPRepSize(icstate.Candidate))
			assert.Equal(t, tt.size, pm.Size())

			err = pm.UnregisterPRep(createAddress(0))
			assert.NoError(t, err)
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
