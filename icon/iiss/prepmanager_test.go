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

func createPRepBase(i int) *icstate.PRepBase {
	owner := createAddress(i)
	node := createAddress(i * 100)
	pb := icstate.NewPRepBase(owner)

	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)
	endpoint := fmt.Sprintf("%s.example.com:9080/api/v3", name)

	err := pb.SetPRep(name, email, website, country, city, details, endpoint, node)
	if err != nil {
		return nil
	}
	return pb
}

func createPRepStatus(i int) *icstate.PRepStatus {
	owner := createAddress(i)
	ps := icstate.NewPRepStatus(owner)
	ps.SetGrade(icstate.Candidate)
	ps.SetStatus(icstate.Active)
	ps.SetDelegated(big.NewInt(rand.Int63()))
	ps.SetBonded(big.NewInt(0))
	return ps
}

// test for GetBondedDelegation
func TestPRepManager_Sort(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	s := icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), false)
	pm := newPRepManager(s, big.NewInt(int64(1000000)))

	addr := common.NewAddressFromString("hx1")
	delegated := big.NewInt(int64(99))
	s.GetPRepStatus(addr, true).SetDelegated(delegated)
	bonded := big.NewInt(int64(2))
	s.GetPRepStatus(addr, true).SetBonded(bonded)
	pm.state.AddActivePRep(addr)

	addr = common.NewAddressFromString("hx2")
	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(1))
	s.GetPRepStatus(addr, true).SetBonded(bonded)
	pm.state.AddActivePRep(addr)

	addr = common.NewAddressFromString("hx3")
	delegated = big.NewInt(int64(99))
	s.GetPRepStatus(addr, true).SetDelegated(delegated)
	bonded = big.NewInt(int64(3))
	s.GetPRepStatus(addr, true).SetBonded(bonded)
	pm.state.AddActivePRep(addr)

	pm.state.SetBondRequirement(5)
	pm.init()
	//	pm.sort()

	assert.Equal(t, "hx0000000000000000000000000000000000000003", pm.orderedPReps[0].Owner().String())
	assert.Equal(t, "hx0000000000000000000000000000000000000001", pm.orderedPReps[1].Owner().String())
	assert.Equal(t, "hx0000000000000000000000000000000000000002", pm.orderedPReps[2].Owner().String())
}

/*
func TestPRepManager_new(t *testing.T) {
	size := 100
	store := createPRepStore(size)

	icobject.NewObjectStoreState()
	pm := newPRepManager(store)
	pm.Reset(store)

	if pm.Size() != size {
		t.Errorf("Size not mismatch: got(%d) != expected(%d)", pm.Size(), size)
	}

	prev := big.NewInt(math.MaxInt64)
	for i := 0; i < size; i++ {
		PRep := pm.GetPRepByIndex(i)
		bondedDelegation := PRep.GetBondedDelegation()

		if prev.Cmp(bondedDelegation) < 0 {
			t.Errorf("PRepManager.sort() is failed")
		}

		prev.Set(bondedDelegation)
	}
}
*/
