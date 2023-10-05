package icsim

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
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
	bonders []module.Address
	preps   []module.Address
	users   []module.Address
	sim     Simulator
}

// init makes the initial environment for penalty and slashing test
func (env *Env) init() error {
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

func (env *Env) SetStakesAll() ([]Receipt, error) {
	var err error
	var receipts []Receipt

	addrsList := [][]module.Address{env.bonders, env.users}
	for _, addrs := range addrsList {
		amount := env.sim.GetBalance(addrs[0])
		receipts, err = env.SetStakes(addrs, amount)
		if !(checkReceipts(receipts) && err == nil) {
			return receipts, err
		}
	}

	return nil, nil
}

func (env *Env) SetStakes(addrs []module.Address, amount *big.Int) ([]Receipt, error) {
	sim := env.sim
	block := NewBlock()
	for _, from := range addrs {
		tx := sim.SetStake(from, amount)
		block.AddTransaction(tx)
	}
	return sim.GoByBlock(nil, block)
}

func (env *Env) SetDelegationsAll() error {
	var err error
	addrsList := [][]module.Address{env.bonders, env.preps, env.users}
	for _, addrs := range addrsList {
		amount, ok := env.sim.GetStake(addrs[0])["stake"].(*big.Int)
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
	block := NewBlock()

	// One user delegates amount of ICX to one prep
	for i, from := range addrs {
		i = i % len(preps)
		prep := preps[i]
		ds := make([]*icstate.Delegation, 1)
		ds[0] = icstate.NewDelegation(common.AddressToPtr(prep), amount)
		block.AddTransaction(sim.SetDelegation(from, ds))
	}
	return sim.GoByBlock(nil, block)
}

func (env *Env) SetBonderLists() ([]Receipt, error) {
	sim := env.sim
	preps := env.preps
	bonders := env.bonders
	block := NewBlock()

	// One user delegates amount of ICX to one prep
	for i, from := range preps {
		bonderList := []*common.Address{
			common.AddressToPtr(from),
			common.AddressToPtr(bonders[i]),
		}
		tx := sim.SetBonderList(from, bonderList)
		block.AddTransaction(tx)
	}
	return sim.GoByBlock(nil, block)
}

func (env *Env) SetBonds() ([]Receipt, error) {
	sim := env.sim
	preps := env.preps
	bonders := env.bonders
	block := NewBlock()

	// One bonder delegates some amount of bond to one prep
	for i, from := range bonders {
		if i >= env.config.BondedPRepCount {
			// Limit the number of bonded preps
			break
		}
		balance := sim.GetBalance(from)
		if balance.Sign() == 0 {
			continue
		}
		bonds := []*icstate.Bond{
			icstate.NewBond(common.AddressToPtr(preps[i]), balance),
		}
		tx := sim.SetBond(from, bonds)
		block.AddTransaction(tx)
	}
	return sim.GoByBlock(nil, block)
}

func (env *Env) Simulator() Simulator {
	return env.sim
}

func NewEnv(c *SimConfig, revision module.Revision) (*Env, error) {
	userLen := 100
	prepLen := int(c.MainPRepCount + c.SubPRepCount)
	bonderLen := prepLen
	validatorLen := int(c.MainPRepCount)

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
	for i, bonder := range bonders {
		if i == 0 {
			balance = icutils.ToLoop(100)
		} else {
			balance = icutils.ToLoop(10)
		}
		balances[icutils.ToKey(bonder)] = balance
	}

	var env *Env
	sim, err := NewSimulator(revision, validators, balances, c)
	if err == nil {
		env = &Env{
			config:  c,
			bonders: bonders,
			preps:   preps,
			users:   users,
			sim:     sim,
		}
		err = env.init()
	}
	return env, err
}
