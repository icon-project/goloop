package icsim

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
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

func newDummyAddresses(start, size int) []module.Address {
	addrs := make([]module.Address, size)
	for i := 0; i < size; i++ {
		addrs[i] = newDummyAddress(start - i)
	}
	return addrs
}

type Env struct {
	config  *SimConfig
	gov     module.Address // Governance
	bso     module.Address // BuiltinScoreOwner
	bonders []module.Address
	preps   []module.Address
	users   []module.Address
	sim     Simulator
}

func (env *Env) Governance() module.Address {
	if env.gov == nil {
		env.gov = common.MustNewAddressFromString("cx0000000000000000000000000000000000000001")
	}
	return env.gov
}

func (env *Env) SetRevision(revision module.Revision) ([]Receipt, error) {
	sim := env.sim
	tx := sim.SetRevision(env.Governance(), revision)
	return sim.GoByTransaction(nil, tx)
}

// RegisterPReps register all preps in Env
func (env *Env) RegisterPReps() ([]Receipt, error) {
	sim := env.sim
	preps := env.preps
	blk := NewBlock()
	for i, from := range preps {
		info := newDummyPRepInfo(i)
		tx := sim.RegisterPRep(from, info)
		blk.AddTransaction(tx)
	}
	return env.sim.GoByBlock(nil, blk)
}

// SetStakesAll stakes a given amount of ICX for bonders and users
func (env *Env) SetStakesAll(amount *big.Int) ([]Receipt, error) {
	var err error
	var receipts []Receipt

	addrsList := [][]module.Address{env.bonders, env.users}
	for _, addrs := range addrsList {
		receipts, err = env.SetStakes(addrs, amount)
		if !(checkReceipts(receipts) && err == nil) {
			return receipts, err
		}
	}

	return nil, nil
}

func (env *Env) SetStakes(addrs []module.Address, amount *big.Int) ([]Receipt, error) {
	sim := env.sim
	blk := NewBlock()
	for _, from := range addrs {
		tx := sim.SetStake(from, amount)
		blk.AddTransaction(tx)
	}
	return sim.GoByBlock(nil, blk)
}

func (env *Env) SetDelegationsAll() error {
	var err error
	addrsList := [][]module.Address{env.bonders, env.preps, env.users}
	for _, addrs := range addrsList {
		amount, ok := env.sim.GetStakeInJSON(addrs[0])["stake"].(*big.Int)
		if ok {
			if _, err = env.SetDelegations(addrs, amount); err != nil {
				return err
			}
		}
	}
	return nil
}

func (env *Env) SetDelegations(addrs []module.Address, amount *big.Int) ([]Receipt, error) {
	sim := env.sim
	preps := env.preps
	blk := NewBlock()

	// One user delegates amount of ICX to one prep
	for i, from := range addrs {
		i = i % len(preps)
		prep := preps[i]
		ds := []*icstate.Delegation{
			icstate.NewDelegation(common.AddressToPtr(prep), amount),
		}
		blk.AddTransaction(sim.SetDelegation(from, ds))
	}
	return sim.GoByBlock(nil, blk)
}

func (env *Env) SetBonderLists() ([]Receipt, error) {
	sim := env.sim
	preps := env.preps
	bonders := env.bonders
	blk := NewBlock()

	// Every PRep has 2 bonders, which are env.bonders[i] and itself
	for i, from := range preps {
		bonderList := []*common.Address{
			common.AddressToPtr(bonders[i]),
			common.AddressToPtr(from),
		}
		tx := sim.SetBonderList(from, bonderList)
		blk.AddTransaction(tx)
	}
	return sim.GoByBlock(nil, blk)
}

// SetBonds makes all env.bonders[i] bond a given amount to env.preps[i]
func (env *Env) SetBonds(amount *big.Int) ([]Receipt, error) {
	sim := env.sim
	preps := env.preps
	bonders := env.bonders
	blk := NewBlock()

	// One bonder delegates a given amount of bond to one prep
	for i, from := range bonders {
		bonds := []*icstate.Bond{
			icstate.NewBond(common.AddressToPtr(preps[i]), amount),
		}
		tx := sim.SetBond(from, bonds)
		blk.AddTransaction(tx)
	}
	return sim.GoByBlock(nil, blk)
}

func (env *Env) Simulator() Simulator {
	return env.sim
}

func NewEnv(c *SimConfig, revision module.Revision) (*Env, error) {
	userLen := 100
	prepLen := int(c.MainPRepCount + c.SubPRepCount)
	bonderLen := prepLen
	validatorLen := int(c.MainPRepCount)

	// Initialize addresses for test
	preps := newDummyAddresses(1000, prepLen)
	users := newDummyAddresses(2000, userLen)
	bonders := newDummyAddresses(3000, bonderLen)

	validators := make([]module.Validator, validatorLen)
	for i := 0; i < validatorLen; i++ {
		addr := newDummyAddress(4000 + i)
		validator, _ := state.ValidatorFromAddress(addr)
		validators[i] = validator
	}

	// Initialize balances for each account
	balances := make(map[string]*big.Int)
	balance := icutils.ToLoop(2000)
	for _, prep := range preps {
		balances[icutils.ToKey(prep)] = balance
	}
	balance = icutils.ToLoop(10000)
	for _, user := range users {
		balances[icutils.ToKey(user)] = balance
	}
	for _, bonder := range bonders {
		balances[icutils.ToKey(bonder)] = balance
	}

	var env *Env
	sim, err := NewSimulator(revision, validators, balances, c)
	if err == nil {
		// Check if balance initialization succeeded
		for k, amount := range balances {
			addr := common.MustNewAddress([]byte(k))
			if amount.Cmp(sim.GetBalance(addr)) != 0 {
				return nil, errors.New("Balance initialization failed")
			}
		}

		env = &Env{
			config:  c,
			bonders: bonders,
			preps:   preps,
			users:   users,
			sim:     sim,
		}
		err = env.init(revision)
	}
	return env, err
}

/*
// init makes the initial environment for penalty and slashing test
func (env *Env) init(revision module.Revision) error {
	var receipts []Receipt
	var err error
	sim := env.sim

	receipts, err = env.RegisterPReps()
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.RegisterPReps()")
	}

	receipts, err = env.SetStakesAll()
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetStakesAll()")
	}

	receipts, err = env.SetDelegations(env.users, icutils.ToLoop(10000))
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetDelegations()")
	}

	receipts, err = env.SetBonderLists()
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetBonderLists()")
	}

	receipts, err = env.SetBonds()
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetBonds()")
	}

	// Activate decentralization
	_ = sim.GoToTermEnd(nil)
	return sim.GoToTermEnd(nil)
}
*/

func (env *Env) init(revision module.Revision) error {
	initHandlers := map[int]func() error{
		icmodule.Revision13: env.initOnRev13,
		icmodule.Revision23: env.initOnRev23,
	}

	targetRev := revision.Value()
	for rev := 0; rev <= targetRev; rev++ {
		if handler, ok := initHandlers[rev]; ok {
			if err := handler(); err != nil {
				return err
			}
		}
	}
	return nil
}

// initOnRev13
// Revision has been already updated in NewSimulator() so revision update is not needed
func (env *Env) initOnRev13() error {
	var err error
	var receipts []Receipt
	sim := env.Simulator()

	//receipts, err = env.SetRevision(icmodule.ValueToRevision(icmodule.Revision13))
	//if err != nil {
	//	return err
	//}
	//if !CheckReceiptSuccess(receipts) {
	//	return errors.New("Receipts Failure")
	//}

	receipts, err = env.RegisterPReps()
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.RegisterPReps()")
	}

	// Stakes 1000 ICX of all bonders and users
	amount := icutils.ToLoop(2000)
	receipts, err = env.SetStakesAll(amount)
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetStakesAll()")
	}

	receipts, err = env.SetDelegations(env.users, amount)
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetDelegations()")
	}

	// Activate decentralization
	if err = sim.GoToTermEnd(nil); err != nil {
		return err
	}

	receipts, err = env.SetBonderLists()
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetBonderLists()")
	}

	receipts, err = env.SetBonds(amount)
	if !(checkReceipts(receipts) && err == nil) {
		return errors.Errorf("Failed to Env.SetBonds()")
	}

	if err = sim.GoToTermEnd(nil); err != nil {
		return err
	}

	// Skip 2 blocks after decentralization
	//return sim.Go(nil, 2)
	return nil
}

// initOnRev23 enables RevisionPreIISS4
func (env *Env) initOnRev23() error {
	var err error
	var receipts []Receipt
	receipts, err = env.SetRevision(icmodule.ValueToRevision(icmodule.Revision23))
	if err != nil {
		return err
	}
	if !CheckReceiptSuccess(receipts...) {
		return errors.New("Receipts Failure")
	}
	return nil
}