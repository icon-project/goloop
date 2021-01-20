package icstate

import (
	"fmt"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"math/big"
)

type PRepSnapshot struct {
	owner            module.Address
	bondedDelegation *big.Int
}

func (pss *PRepSnapshot) Owner() module.Address {
	return pss.owner
}

func (pss *PRepSnapshot) BondedDelegation() *big.Int {
	return pss.bondedDelegation
}

func (pss *PRepSnapshot) Equal(other *PRepSnapshot) bool {
	if pss == other {
		return true
	}
	if pss == nil || other == nil {
		return false
	}
	return pss.owner.Equal(other.owner) &&
		pss.bondedDelegation.Cmp(other.bondedDelegation) == 0
}

func (pss *PRepSnapshot) Clone() *PRepSnapshot {
	return &PRepSnapshot{
		owner:            pss.owner,
		bondedDelegation: new(big.Int).Set(pss.bondedDelegation),
	}
}

func (pss *PRepSnapshot) toJSON() map[string]interface{} {
	jso := make(map[string]interface{}, 2)
	jso["address"] = pss.owner.String()
	jso["bondedDelegation"] = pss.bondedDelegation
	return jso
}

func (pss *PRepSnapshot) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(pss.owner, pss.bondedDelegation)
}

func (pss *PRepSnapshot) RLPDecodeSelf(d codec.Decoder) error {
	var owner *common.Address
	err := d.DecodeListOf(&owner, &pss.bondedDelegation)
	if err == nil {
		pss.owner = owner
	}

	return err
}

func NewPRepSnapshotFromPRepStatus(ps *PRepStatus, bondRequirement int) *PRepSnapshot {
	return &PRepSnapshot{
		owner:            ps.owner,
		bondedDelegation: new(big.Int).Set(ps.GetBondedDelegation(bondRequirement)),
	}
}

type PRepSnapshots []*PRepSnapshot

func (p PRepSnapshots) Equal(other PRepSnapshots) bool {
	if len(p) != len(other) {
		return false
	}

	for i := range p {
		if !p[i].Equal(other[i]) {
			return false
		}
	}
	return true
}

func (p PRepSnapshots) Clone() PRepSnapshots {
	if p == nil {
		return nil
	}

	size := len(p)
	ret := make(PRepSnapshots, size, size)
	for i := 0; i < size; i++ {
		ret[i] = p[i].Clone()
	}
	return ret
}

func (p PRepSnapshots) toJSON() []interface{} {
	size := len(p)
	jso := make([]interface{}, size, size)

	for i, pss := range p {
		jso[i] = pss.toJSON()
	}

	return jso
}

type TermFlag int

const (
	FlagNextTerm TermFlag = 1 << iota
	FlagValidator

	FlagNone TermFlag = 0
	FlagAll  TermFlag = 0xFFFFFFFF
)

type Term struct {
	icobject.NoDatabase
	StateAndSnapshot

	sequence              int
	startHeight           int64
	period                int64
	irep                  *big.Int
	rrep                  *big.Int
	totalSupply           *big.Int
	totalDelegated        *big.Int // total delegated amount of all active P-Reps. Set with PRepManager.totalDelegated
	prepSnapshots         PRepSnapshots

	flags       TermFlag
	snapshotMap map[string]*PRepSnapshot
}

func (term *Term) GetEndBlockHeight() int64 {
	if term == nil {
		return -1
	}
	return term.startHeight + term.period - 1
}

func (term *Term) Set(other *Term) {
	term.checkWritable()
	term.sequence = other.sequence
	term.startHeight = other.startHeight
	term.period = other.period
	term.irep = other.irep
	term.rrep = other.rrep
	term.totalSupply.Set(other.totalSupply)
	term.totalDelegated.Set(other.totalDelegated)
	term.SetPRepSnapshots(other.prepSnapshots.Clone())
	term.flags = FlagNone
}

func (term *Term) Clone() *Term {
	if term == nil {
		return nil
	}

	return &Term{
		sequence:              term.sequence,
		startHeight:           term.startHeight,
		period:                term.period,
		irep:                  new(big.Int).Set(term.irep),
		rrep:                  new(big.Int).Set(term.rrep),
		totalSupply:           new(big.Int).Set(term.totalSupply),
		totalDelegated:        new(big.Int).Set(term.totalDelegated),
		prepSnapshots:         term.prepSnapshots.Clone(),
	}
}

func (term *Term) Version() int {
	return 0
}

func (term *Term) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeListOf(
		&term.sequence,
		&term.startHeight,
		&term.period,
		&term.irep,
		&term.rrep,
		&term.totalSupply,
		&term.totalDelegated,
		&term.prepSnapshots,
	)
}

func (term *Term) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeListOf(
		term.sequence,
		term.startHeight,
		term.period,
		term.irep,
		term.rrep,
		term.totalSupply,
		term.totalDelegated,
		term.prepSnapshots,
	)
}

func (term *Term) Equal(o icobject.Impl) bool {
	if other, ok := o.(*Term); ok {
		return term.equal(other)
	}
	return false
}

func (term *Term) equal(other *Term) bool {
	if term == other {
		return true
	}
	if term == nil || other == nil {
		return false
	}

	return term.sequence == other.sequence &&
		term.startHeight == other.startHeight &&
		term.period == other.period &&
		term.irep.Cmp(other.irep) == 0 &&
		term.rrep.Cmp(other.rrep) == 0 &&
		term.totalSupply.Cmp(other.totalSupply) == 0 &&
		term.totalDelegated.Cmp(other.totalDelegated) == 0 &&
		term.prepSnapshots.Equal(other.prepSnapshots)
}

func (term *Term) GetPRepSnapshotCount() int {
	return len(term.prepSnapshots)
}

func (term *Term) GetPRepSnapshotByIndex(index int) *PRepSnapshot {
	if index < 0 || index >= term.GetPRepSnapshotCount() {
		return nil
	}
	return term.prepSnapshots[index]
}

func (term *Term) GetPRepSnapshotByOwner(owner module.Address) *PRepSnapshot {
	if term.snapshotMap == nil {
		return nil
	}
	return term.snapshotMap[icutils.ToKey(owner)]
}

func (term *Term) getPRepSnapshotIndex(owner module.Address) int {
	ps := term.snapshotMap[icutils.ToKey(owner)]
	if ps != nil {
		size := len(term.prepSnapshots)
		for i := 0; i < size; i++ {
			if owner.Equal(term.prepSnapshots[i].owner) {
				return i
			}
		}
	}
	return -1
}

func (term *Term) IsUpdated() bool {
	return term.flags != FlagNone
}

func (term *Term) IsAnyFlagOn(flags TermFlag) bool {
	return term.flags&flags != FlagNone
}

func (term *Term) GetFlag() TermFlag {
	return term.flags
}

func (term *Term) ResetFlag() {
	term.flags = FlagNone
}

func (term *Term) SetFlag(flags TermFlag, on bool) {
	if on {
		term.flags |= flags
	} else {
		term.flags &= ^flags
	}
}

func (term *Term) TotalDelegated() *big.Int {
	return term.totalDelegated
}

func (term *Term) GetTotalBondedDelegation() *big.Int {
	totalBondedDelegation := new(big.Int)
	if term.prepSnapshots != nil {
		for _, ps := range term.prepSnapshots {
			totalBondedDelegation.Add(totalBondedDelegation, ps.bondedDelegation)
		}
	}

	return totalBondedDelegation
}

func (term *Term) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})

	jso["sequence"] = term.sequence
	jso["startBlockHeight"] = term.startHeight
	jso["endBlockHeight"] = term.GetEndBlockHeight()
	jso["totalSupply"] = term.totalSupply
	jso["totalDelegated"] = term.totalDelegated
	jso["totalBondedDelegation"] = term.GetTotalBondedDelegation()
	jso["irep"] = term.irep
	jso["rrep"] = term.rrep
	jso["period"] = term.period
	jso["preps"] = term.prepSnapshots.toJSON()

	return jso
}

func (term *Term) NewNextTerm(s *State, totalSupply *big.Int, totalDelegated *big.Int) (*Term, error) {
	nextTerm := &Term{
		sequence:              term.sequence + 1,
		startHeight:           term.GetEndBlockHeight() + 1,
		period:                GetTermPeriod(s.store),
		irep:                  new(big.Int).Set(GetIRep(s)),
		rrep:                  new(big.Int).Set(GetRRep(s)),
		totalSupply:           new(big.Int).Set(totalSupply),
		totalDelegated:        new(big.Int).Set(totalDelegated),

		flags: term.flags | FlagNextTerm,
	}
	return nextTerm, nil
}

func (term *Term) GetSnapshot(store *icobject.ObjectStoreState) error {
	if !term.IsAnyFlagOn(FlagAll) {
		return nil
	}
	o := icobject.New(TypeTerm, term)
	varDB := containerdb.NewVarDB(store, termVarPrefix)
	return varDB.Set(o)
}

func (term *Term) RemovePRepSnapshot(owner module.Address) error {
	if term == nil {
		return errors.Errorf("Term is nil")
	}

	key := icutils.ToKey(owner)
	ps := term.snapshotMap[key]
	if ps == nil {
		return errors.Errorf("PRepSnapshot not found: %s", owner)
	}

	// Remove prepSnapshot from slice
	idx := term.getPRepSnapshotIndex(owner)
	if err := term.removePRepSnapshotByIndex(idx); err != nil {
		return err
	}

	// Remove prepSnapshot from map
	delete(term.snapshotMap, key)
	return nil
}

func (term *Term) removePRepSnapshotByIndex(idx int) error {
	prepSnapshots := term.prepSnapshots
	size := len(prepSnapshots)

	if idx < 0 || idx >= size {
		return errors.Errorf("Index out of range")
	}

	for i := idx + 1; i < size; i++ {
		prepSnapshots[i-1] = prepSnapshots[i]
	}

	term.prepSnapshots = prepSnapshots[:size-1]
	return nil
}

func (term *Term) SetPRepSnapshots(prepSnapshots []*PRepSnapshot) {
	var snapshotMap map[string]*PRepSnapshot = nil
	term.prepSnapshots = prepSnapshots

	if prepSnapshots != nil {
		snapshotMap = make(map[string]*PRepSnapshot)
		for _, ps := range prepSnapshots {
			key := icutils.ToKey(ps.owner)
			snapshotMap[key] = ps
		}
	}

	term.snapshotMap = snapshotMap
	term.flags |= FlagValidator
}

func (term *Term) String() string {
	return fmt.Sprintf(
		"Term: seq=%d startBH=%d endBH=%d period=%d totalSupply=%s totalDelegated=%s preps=%d",
		term.sequence,
		term.startHeight,
		term.GetEndBlockHeight(),
		term.period,
		term.totalSupply,
		term.totalDelegated,
		len(term.prepSnapshots),
	)
}

func (term *Term) IsDecentralized() bool {
	if term == nil {
		return false
	}
	return len(term.prepSnapshots) > 0
}

func newTermWithTag(_ icobject.Tag) *Term {
	return &Term{}
}

func newTerm(termPeriod int64) *Term {
	return &Term{
		period:                termPeriod,
		irep:                  big.NewInt(0),
		rrep:                  big.NewInt(0),
		totalSupply:           big.NewInt(0),
		totalDelegated:        big.NewInt(0),
		prepSnapshots:         nil,

		flags:       FlagNone,
		snapshotMap: nil,
	}
}
