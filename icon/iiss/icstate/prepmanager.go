package icstate

import (
	"bytes"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
	"math/big"
	"sort"
)

const (
	mainPRepCount = 22
	subPRepCount  = 78
)

type RegInfoIdx int

func (_ RegInfoIdx) Size() int {
	return idxSize
}

const (
	IdxCity RegInfoIdx = iota
	IdxCountry
	IdxDetails
	IdxEmail
	IdxName
	IdxP2pEndpoint
	IdxWebsite

	idxSize = iota - 1
)

var (
	activePRepArrayPrefix = containerdb.ToKey(containerdb.RawBuilder, "active_prep")
	nodeOwnerDictPrefix   = containerdb.ToKey(containerdb.RawBuilder, "node_owner")
	prepBaseDictPrefix    = containerdb.ToKey(containerdb.RawBuilder, "prep_base")
	prepStatusDictPrefix  = containerdb.ToKey(containerdb.RawBuilder, "prep_status")
)

type PRep struct {
	owner module.Address
	*PRepBase
	*PRepStatus
}

func (p *PRep) Owner() module.Address {
	return p.owner
}

func (p *PRep) ToJSON() map[string]interface{} {
	return icutils.MergeMaps(p.PRepBase.ToJSON(), p.PRepStatus.ToJSON())
}

func (p *PRep) Clone() *PRep {
	return newPRep(p.owner, p.PRepBase, p.PRepStatus)
}

func newPRep(owner module.Address, base *PRepBase, status *PRepStatus) *PRep {
	base = base.Clone()
	base.SetOwner(owner)

	status = status.Clone()
	status.SetOwner(owner)

	return &PRep{owner: owner, PRepBase: base, PRepStatus: status}
}

func setPRep(pb *PRepBase, node module.Address, params []string) error {
	return pb.SetPRep(
		params[IdxName],
		params[IdxEmail],
		params[IdxWebsite],
		params[IdxCountry],
		params[IdxCity],
		params[IdxDetails],
		params[IdxP2pEndpoint],
		node,
	)
}

// Manage PRepBase, PRepStatus and ActivePRep
type PRepManager struct {
	totalDelegated  *big.Int
	totalStake      *big.Int
	store           containerdb.ObjectStoreState
	activePRepCache *ActivePRepCache
	nodeOwnerCache  *NodeOwnerCache
	prepBaseCache   *PRepBaseCache
	prepStatusCache *PRepStatusCache

	orderedPReps []*PRep
	prepMap      map[string]*PRep
}

type prepFulls []*PRep

func (p prepFulls) Len() int      { return len(p) }
func (p prepFulls) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p prepFulls) Less(i, j int) bool {
	ret := p[i].GetBondedDelegation().Cmp(p[j].GetBondedDelegation())
	if ret < 0 {
		return true
	} else if ret > 0 {
		return false
	}

	ret = p[i].Delegated().Cmp(p[j].Delegated())
	if ret < 0 {
		return true
	} else if ret > 0 {
		return false
	}

	return bytes.Compare(p[i].GetNode().Bytes(), p[j].GetNode().Bytes()) < 0
}

func (pm *PRepManager) init() {
	pm.activePRepCache.Reset()
	size := pm.activePRepCache.Size()

	for i := 0; i < size; i++ {
		owner := pm.activePRepCache.Get(i)
		prep := pm.getPRep(owner)
		pm.Add(prep)
	}

	pm.sort()
}

func (pm *PRepManager) getPRepBase(owner module.Address) *PRepBase {
	return pm.prepBaseCache.Get(owner)
}

func (pm *PRepManager) getPRepStatus(owner module.Address) *PRepStatus {
	return pm.prepStatusCache.Get(owner)
}

func (pm *PRepManager) getPRep(owner module.Address) *PRep {
	base := pm.getPRepBase(owner)
	if base == nil {
		return nil
	}

	status := pm.getPRepStatus(owner)
	return newPRep(owner, base, status)
}

func (pm *PRepManager) Add(p *PRep) {
	pm.orderedPReps = append(pm.orderedPReps, p)
	pm.prepMap[icutils.ToKey(p.Owner())] = p
	pm.totalDelegated.Add(pm.totalDelegated, p.Delegated())
}

// sort prepFulls in descending order by bonded delegation
func (pm *PRepManager) sort() {
	sort.Sort(sort.Reverse(prepFulls(pm.orderedPReps)))
}

func (pm *PRepManager) Size() int {
	return len(pm.orderedPReps)
}

func (pm *PRepManager) GetPRepByOwner(owner module.Address) *PRep {
	return pm.prepMap[icutils.ToKey(owner)]
}

func (pm *PRepManager) GetPRepByNode(node module.Address) *PRep {
	owner := pm.nodeOwnerCache.Get(node)
	if owner == nil {
		owner = node
	}

	return pm.GetPRepByOwner(owner)
}

func (pm *PRepManager) GetPRepByIndex(i int) *PRep {
	return pm.orderedPReps[i]
}

func (pm *PRepManager) TotalDelegated() *big.Int {
	return pm.totalDelegated
}

func (pm *PRepManager) TotalStake() *big.Int {
	// TODO: Not implemented
	return pm.totalStake
}

func (pm *PRepManager) GetValidators() []module.Validator {
	size := len(pm.orderedPReps)
	if size < mainPRepCount {
		log.Errorf("Not enough PReps: %d < %d", size, mainPRepCount)
	}

	var err error
	var address module.Address
	validators := make([]module.Validator, size)
	for i := 0; i < size; i++ {
		address = pm.orderedPReps[i].GetNode()
		validators[i], err = state.ValidatorFromAddress(address)
		if err != nil {
			log.Errorf("Failed to run GetValidators(): %s", address.String())
		}
	}

	return validators
}

func (pm *PRepManager) GetPRepsInJSON() map[string]interface{} {
	size := len(pm.orderedPReps)
	ret := make(map[string]interface{})
	preps := make([]map[string]interface{}, size, size)
	ret["preps"] = preps

	for i, prep := range pm.orderedPReps {
		preps[i] = prep.ToJSON()
	}

	return ret
}

func (pm *PRepManager) Reset() error {
	pm.activePRepCache.Reset()
	pm.prepBaseCache.Reset()
	pm.prepStatusCache.Reset()
	pm.nodeOwnerCache.Reset()
	return nil
}

// It is called on ExtensionState.GetSnapshot()
func (pm *PRepManager) GetSnapshot() error {
	pm.activePRepCache.GetSnapshot()
	pm.prepBaseCache.GetSnapshot()
	pm.prepStatusCache.GetSnapshot()
	pm.nodeOwnerCache.GetSnapshot()
	return nil
}

func (pm *PRepManager) contains(owner module.Address) bool {
	pb := pm.getPRepBase(owner)
	return !pb.IsEmpty()
}

func (pm *PRepManager) RegisterPRep(owner, node module.Address, params []string) error {
	if pm.contains(owner) {
		return errors.Errorf("PRep already exists: %s", owner)
	}

	pb := newPRepBase(owner)
	err := setPRep(pb, node, params)
	if err != nil {
		return err
	}

	ps := newPRepStatus(owner)
	pm.prepBaseCache.Add(pb)
	pm.prepStatusCache.Add(ps)
	pm.activePRepCache.Add(owner)
	if err = pm.addNodeToOwner(node, owner); err != nil {
		return err
	}

	return nil
}

func (pm *PRepManager) SetPRep(owner, node module.Address, params []string) error {
	pb := pm.getPRepBase(owner)
	if pb == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}
	return setPRep(pb, node, params)
}

func (pm *PRepManager) UnregisterPRep(owner module.Address) error {
	var err error
	p := pm.getPRep(owner)
	if p == nil {
		return errors.Errorf("PRep not found: %s", owner)
	}

	err = pm.prepBaseCache.Remove(owner)
	if err != nil {
		return err
	}
	err = pm.prepStatusCache.Remove(owner)
	if err != nil {
		return err
	}
	pm.totalDelegated.Sub(pm.totalDelegated, p.delegated)
	return nil
}

func (pm *PRepManager) addNodeToOwner(node, owner module.Address) error {
	if node == nil {
		return nil
	}
	if owner.Equal(node) {
		return nil
	}
	if pm.contains(node) {
		return errors.Errorf("Node must not be owner of other")
	}
	return pm.nodeOwnerCache.Add(node, owner)
}

func (pm *PRepManager) ChangeDelegation(od, nd Delegations) error {
	delta := make(map[string]*big.Int)

	for _, d := range od {
		key := icutils.ToKey(d.Address)
		delta[key] = new(big.Int).Neg(d.Value.Value())
	}
	for _, d := range nd {
		key := icutils.ToKey(d.Address)
		if delta[key] == nil {
			delta[key] = new(big.Int)
		}
		delta[key].Add(delta[key], d.Value.Value())
	}

	for k, v := range delta {
		owner, err := common.NewAddress([]byte(k))
		if err != nil {
			return err
		}

		ps := pm.prepStatusCache.Get(owner)
		if ps == nil {
			return errors.Errorf("PRepStatus is not found: %s", owner)
		}

		key := icutils.ToKey(owner)
		if delta[key].Cmp(BigIntZero) != 0 {
			newPs := ps.Clone()
			newPs.delegated.Add(newPs.delegated, v)
			pm.prepStatusCache.Add(newPs)
		}
	}

	pm.totalDelegated.Add(pm.totalDelegated, nd.GetDelegationAmount()).Sub(pm.totalDelegated, od.GetDelegationAmount())
	return nil
}

func newPRepManager(store containerdb.ObjectStoreState, totalStake *big.Int) *PRepManager {
	pm := &PRepManager{
		totalDelegated:  big.NewInt(0),
		totalStake:      totalStake,
		activePRepCache: newActivePRepCache(store),
		nodeOwnerCache:  newNodeOwnerCache(store),
		prepBaseCache:   newPRepBaseCache(store),
		prepStatusCache: newPRepStatusCache(store),

		prepMap: make(map[string]*PRep),
	}

	pm.init()
	return pm
}
