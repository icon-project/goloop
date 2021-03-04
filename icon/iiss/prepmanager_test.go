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
	pm := newPRepManager(s)

	for i := 0; i < size; i++ {
		assert.NoError(t, pm.RegisterPRep(newRegInfo(i)))
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

// test for GetBondedDelegation
func TestPRepManager_Sort(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	s := icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), false)
	pm := newPRepManager(s)

	addr := common.MustNewAddressFromString("hx1")
	delegated := big.NewInt(int64(99))
	s.GetPRepBase(addr, true)
	s.GetPRepStatus(addr, true).SetDelegated(delegated)
	bonded := big.NewInt(int64(2))
	s.GetPRepStatus(addr, true).SetBonded(bonded)
	pm.state.AddActivePRep(addr)

	addr = common.MustNewAddressFromString("hx2")
	s.GetPRepBase(addr, true)
	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(1))
	s.GetPRepStatus(addr, true).SetBonded(bonded)
	pm.state.AddActivePRep(addr)

	addr = common.MustNewAddressFromString("hx3")
	s.GetPRepBase(addr, true)
	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(3))
	s.GetPRepStatus(addr, true).SetBonded(bonded)
	pm.state.AddActivePRep(addr)

	assert.NoError(t, pm.state.SetBondRequirement(5))
	pm.init()
	pm.Sort()

	assert.Equal(t, "hx0000000000000000000000000000000000000003", pm.orderedPReps[0].Owner().String())
	assert.Equal(t, "hx0000000000000000000000000000000000000001", pm.orderedPReps[1].Owner().String())
	assert.Equal(t, "hx0000000000000000000000000000000000000002", pm.orderedPReps[2].Owner().String())
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

		err := pm.RegisterPRep(regInfo)
		assert.NoError(t, err)
		assert.Equal(t, i+1, pm.Size())

		owner = createAddress(i)
		prep := pm.GetPRepByOwner(owner)
		assert.Equal(t, icstate.Candidate, prep.Grade())
		assert.Equal(t, icstate.Active, prep.Status())
		assert.True(t, compareRegInfo(prep, regInfo))

		pb := pm.state.GetPRepBase(owner, false)
		assert.True(t, pb == prep.PRepBase)
		ps := pm.state.GetPRepStatus(owner, false)
		assert.True(t, ps == prep.PRepStatus)
	}

	assert.True(t, checkOrderedByBondedDelegation(pm, 5))
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
		assert.Nil(t, noPRep)

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
		name := fmt.Sprintf("test-%d", i)
		t.Run(name, func(t *testing.T) {
			pm := createPRepManager(t, false, tt.size)
			err := pm.OnTermEnd(tt.mainPRepCount, tt.subPRepCount)
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

func TestPRepManager_UpdateLastState(t *testing.T) {
	type input struct {
		ls     icstate.ValidationState
		lh     int64
		vFail  int64
		vTotal int64

		voted bool
		bh    int64
	}
	type output struct {
		ls        icstate.ValidationState
		lh        int64
		vFail     int64
		vFailCont int64
		vTotal    int64
	}
	type test struct {
		in  input
		out output
	}

	owner := createAddress(0)
	tests := [...]test{
		{
			in: input{
				ls:     icstate.None,
				lh:     100,
				vFail:  10,
				vTotal: 999,

				voted: false,
				bh:    123,
			},
			out: output{
				ls:        icstate.None,
				lh:        123,
				vFail:     11,
				vFailCont: 0,
				vTotal:    1000,
			},
		},
		{
			in: input{
				ls:     icstate.None,
				lh:     100,
				vFail:  123,
				vTotal: 9999,

				voted: true,
				bh:    110,
			},
			out: output{
				ls:        icstate.None,
				lh:        110,
				vFail:     123,
				vFailCont: 0,
				vTotal:    10000,
			},
		},
		{
			in: input{
				ls:     icstate.Failure,
				lh:     1000,
				vFail:  51,
				vTotal: 90,

				voted: true,
				bh:    1010,
			},
			out: output{
				ls:        icstate.Success,
				lh:        1010,
				vFail:     60,
				vFailCont: 0,
				vTotal:    100,
			},
		},
		{
			in: input{
				ls:     icstate.Success,
				lh:     1000,
				vFail:  50,
				vTotal: 90,

				voted: false,
				bh:    1010,
			},
			out: output{
				ls:        icstate.Failure,
				lh:        1010,
				vFail:     51,
				vFailCont: 1,
				vTotal:    100,
			},
		},
		{
			in: input{
				ls:     icstate.Failure,
				lh:     1000,
				vFail:  50,
				vTotal: 90,

				voted: false,
				bh:    1010,
			},
			out: output{
				ls:        icstate.Failure,
				lh:        1000,
				vFail:     60,
				vFailCont: 11,
				vTotal:    100,
			},
		},
		{
			in: input{
				ls:     icstate.Success,
				lh:     1000,
				vFail:  50,
				vTotal: 90,

				voted: true,
				bh:    1010,
			},
			out: output{
				ls:        icstate.Success,
				lh:        1000,
				vFail:     50,
				vFailCont: 0,
				vTotal:    100,
			},
		},
	}

	for i, tc := range tests {
		name := fmt.Sprintf("test-%d", i)
		t.Run(name, func(t *testing.T) {
			in := tc.in
			out := tc.out
			voted := in.voted
			bh := in.bh

			pm := createPRepManager(t, false, 1)
			prep := pm.GetPRepByOwner(owner)

			// Initialize PRep
			prep.SetLastState(in.ls)
			prep.SetLastHeight(in.lh)
			prep.SetVTotal(in.vTotal)
			prep.SetVFail(in.vFail)

			// Run the method to test
			err := pm.UpdateLastState(owner, voted, bh)
			assert.NoError(t, err)

			// Check the result
			assert.Equal(t, out.ls, prep.LastState())
			assert.Equal(t, out.lh, prep.LastHeight())
			assert.Equal(t, out.vFail, prep.GetVFail(bh))
			assert.Equal(t, out.vTotal, prep.GetVTotal(bh))
			assert.Equal(t, out.vFailCont, prep.GetVFailCont(bh))
		})
	}
}
