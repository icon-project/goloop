package icstate

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"math/big"
	"math/rand"
	"testing"
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

func createPRep(i int) *PRepBase {
	city := fmt.Sprintf("Seoul%d", i)
	country := "KOR"
	name := fmt.Sprintf("node%d", i)
	email := fmt.Sprintf("%s@email.com", name)
	website := fmt.Sprintf("https://%s.example.com/", name)
	details := fmt.Sprintf("%sdetails/", website)

	return &PRepBase{
		city:    city,
		country: country,
		details: details,
		email:   email,
		name:    name,
		website: website,
	}
}

func createPRepStatus(i int) *PRepStatus {
	return &PRepStatus{
		grade:     Candidate,
		status:    Active,
		penalty:   0,
		delegated: big.NewInt(rand.Int63()),
		bonded:    big.NewInt(int64(i * 10)),
	}
}

type PRepStoreImpl struct {
	activePReps []module.Address
	prepMap     map[string]*PRep
}

func (psi *PRepStoreImpl) add(owner module.Address, base *PRepBase, status *PRepStatus) {
	psi.activePReps = append(psi.activePReps, owner)
	psi.prepMap[owner.String()] = newPRep(owner, base, status)
}

func (psi *PRepStoreImpl) GetActivePRepSize() int {
	return len(psi.activePReps)
}

func (psi *PRepStoreImpl) GetActivePRep(i int) module.Address {
	return psi.activePReps[i]
}

func (psi *PRepStoreImpl) GetPRep(owner module.Address) *PRepBase {
	prepFull, ok := psi.prepMap[owner.String()]
	if !ok {
		return nil
	}
	return prepFull.PRepBase
}

func (psi *PRepStoreImpl) GetPRepStatus(owner module.Address) *PRepStatus {
	prepFull, ok := psi.prepMap[owner.String()]
	if !ok {
		return nil
	}
	return prepFull.PRepStatus
}

func (psi *PRepStoreImpl) getPRep(i int) *PRep {
	owner := psi.activePReps[i]
	return psi.prepMap[owner.String()]
}

func createPRepStore(size int) *PRepStoreImpl {
	psi := &PRepStoreImpl{
		activePReps: make([]module.Address, 0, size),
		prepMap:     make(map[string]*PRep),
	}

	for i := 0; i < size; i++ {
		owner := createAddress(i)
		p := createPRep(i)
		ps := createPRepStatus(i)
		psi.add(owner, p, ps)
	}

	return psi
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

func TestPRep_ToJSON(t *testing.T) {
	store := createPRepStore(10)
	prepFull := store.getPRep(5)
	fmt.Printf("%#v", prepFull.ToJSON())
}
