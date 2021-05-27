package icstate

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
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
		bondedDelegation: pss.bondedDelegation,
	}
}

func (pss *PRepSnapshot) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})
	jso["address"] = pss.owner
	jso["bondedDelegation"] = pss.bondedDelegation
	jso["delegated"] = pss.bondedDelegation
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

func NewPRepSnapshotFromPRepStatus(owner module.Address, ps *PRepStatus, bondRequirement int64) *PRepSnapshot {
	return &PRepSnapshot{
		owner:            owner,
		bondedDelegation: ps.GetBondedDelegation(bondRequirement),
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
		jso[i] = pss.ToJSON()
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

	sequence        int
	startHeight     int64
	period          int64
	irep            *big.Int
	rrep            *big.Int
	totalSupply     *big.Int
	totalDelegated  *big.Int // total delegated amount of all active P-Reps. Set with PRepManager.totalDelegated
	rewardFund      *RewardFund
	bondRequirement int
	revision        int
	mainPRepCount   int
	prepSnapshots   PRepSnapshots

	flags       TermFlag
	snapshotMap map[string]*PRepSnapshot
}

func (term *Term) Sequence() int {
	return term.sequence
}

func (term *Term) ResetSequence() {
	term.checkWritable()
	term.sequence = 0
}

func (term *Term) StartHeight() int64 {
	return term.startHeight
}

func (term *Term) Period() int64 {
	return term.period
}

func (term *Term) Irep() *big.Int {
	return term.irep
}

func (term *Term) Rrep() *big.Int {
	return term.rrep
}

func (term *Term) MainPRepCount() int {
	return term.mainPRepCount
}

func (term *Term) GetElectedPRepCount() int {
	return len(term.prepSnapshots)
}

func (term *Term) RewardFund() *RewardFund {
	return term.rewardFund
}

func (term *Term) Iglobal() *big.Int {
	return term.rewardFund.Iglobal
}

func (term *Term) Iprep() *big.Int {
	return term.rewardFund.Iprep
}

func (term *Term) Icps() *big.Int {
	return term.rewardFund.Icps
}

func (term *Term) Irelay() *big.Int {
	return term.rewardFund.Irelay
}

func (term *Term) Ivoter() *big.Int {
	return term.rewardFund.Ivoter
}

func (term *Term) BondRequirement() int {
	return term.bondRequirement
}

func (term *Term) Revision() int {
	return term.revision
}

func (term *Term) GetEndBlockHeight() int64 {
	if term == nil {
		return -1
	}
	return term.startHeight + term.period - 1
}

func (term *Term) GetIISSVersion() int {
	if term.revision >= icmodule.RevisionICON2 {
		return IISSVersion2
	}
	if term.revision >= icmodule.RevisionIISS {
		return IISSVersion1
	}
	return IISSVersion1
}

func (term *Term) VoteStartHeight() int64 {
	if term.sequence == 0 && term.GetIISSVersion() == IISSVersion2 {
		// It's decentralized in test network under GOLOOP
		return term.startHeight + 2
	} else {
		// It's decentralized in main network under LOOPCHAIN
		return term.startHeight + 1
	}
}

func (term *Term) Set(other *Term) {
	term.checkWritable()
	term.sequence = other.sequence
	term.startHeight = other.startHeight
	term.period = other.period
	term.irep = other.irep
	term.rrep = other.rrep
	term.totalSupply = other.totalSupply
	term.totalDelegated = other.totalDelegated
	term.rewardFund = other.rewardFund.Clone()
	term.bondRequirement = other.bondRequirement
	term.revision = other.revision
	term.mainPRepCount = other.mainPRepCount
	term.SetPRepSnapshots(other.prepSnapshots.Clone())
	term.flags = FlagNone
}

func (term *Term) Clone() *Term {
	if term == nil {
		return nil
	}

	return &Term{
		sequence:        term.sequence,
		startHeight:     term.startHeight,
		period:          term.period,
		irep:            term.irep,
		rrep:            term.rrep,
		totalSupply:     term.totalSupply,
		totalDelegated:  term.totalDelegated,
		rewardFund:      term.rewardFund.Clone(),
		bondRequirement: term.bondRequirement,
		revision:        term.revision,
		mainPRepCount:   term.mainPRepCount,
		prepSnapshots:   term.prepSnapshots.Clone(),
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
		&term.rewardFund,
		&term.bondRequirement,
		&term.revision,
		&term.mainPRepCount,
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
		term.rewardFund,
		term.bondRequirement,
		term.revision,
		term.mainPRepCount,
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
		term.rewardFund.Equal(other.rewardFund) &&
		term.bondRequirement == other.bondRequirement &&
		term.revision == other.revision &&
		term.mainPRepCount == other.mainPRepCount &&
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
			if owner.Equal(term.prepSnapshots[i].Owner()) {
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

func (term *Term) TotalSupply() *big.Int {
	return term.totalSupply
}

func (term *Term) TotalDelegated() *big.Int {
	return term.totalDelegated
}

func (term *Term) GetTotalBondedDelegation() *big.Int {
	totalBondedDelegation := new(big.Int)
	if term.prepSnapshots != nil {
		for _, ps := range term.prepSnapshots {
			totalBondedDelegation.Add(totalBondedDelegation, ps.BondedDelegation())
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
	jso["rewardFund"] = term.rewardFund.ToJSON()
	jso["bondRequirement"] = term.bondRequirement
	jso["revision"] = term.revision
	jso["mainPRepCount"] = term.mainPRepCount
	jso["iissVersion"] = term.GetIISSVersion()
	jso["preps"] = term.prepSnapshots.toJSON()

	return jso
}

func NewNextTerm(
	term *Term,
	period int64,
	irep *big.Int,
	rrep *big.Int,
	totalSupply *big.Int,
	totalDelegated *big.Int,
	rewardFund *RewardFund,
	bondRequirement int,
	revision int,
) *Term {
	if term == nil {
		return nil
	}
	nextTerm := &Term{
		sequence:        term.sequence + 1,
		startHeight:     term.GetEndBlockHeight() + 1,
		period:          period,
		irep:            irep,
		rrep:            rrep,
		totalSupply:     totalSupply,
		totalDelegated:  totalDelegated,
		rewardFund:      rewardFund.Clone(),
		bondRequirement: bondRequirement,
		revision:        revision,

		flags: term.flags | FlagNextTerm,
	}
	return nextTerm
}

func GenesisTerm(
	state *State,
	startHeight int64,
	revision int,
) *Term {
	return &Term{
		sequence:        0,
		startHeight:     startHeight,
		period:          state.GetTermPeriod(),
		irep:            state.GetIRep(),
		rrep:            state.GetRRep(),
		totalSupply:     new(big.Int),
		totalDelegated:  new(big.Int),
		rewardFund:      state.GetRewardFund().Clone(),
		bondRequirement: int(state.GetBondRequirement()),
		revision:        revision,

		flags: FlagNextTerm,
	}
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
	term.checkWritable()
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
	term.checkWritable()
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

func (term *Term) SetMainPRepCount(mainPRepCount int) {
	term.checkWritable()
	term.mainPRepCount = mainPRepCount
}

func (term *Term) SetIrep(irep *big.Int) {
	term.checkWritable()
	term.irep = irep
}

func (term *Term) SetRrep(rrep *big.Int) {
	term.checkWritable()
	term.rrep = rrep
}

func (term *Term) String() string {
	return fmt.Sprintf(
		"Term: seq=%d start=%d end=%d period=%d ts=%s td=%s pss=%d irep:%s rrep:%s",
		term.sequence,
		term.startHeight,
		term.GetEndBlockHeight(),
		term.period,
		term.totalSupply,
		term.totalDelegated,
		len(term.prepSnapshots),
		term.irep,
		term.rrep,
	)
}

func (term *Term) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"Term{seq=%d start=%d end=%d period=%d totalSupply=%s totalDelegated=%s "+
					"prepSnapshot=%d irep=%s rrep=%s}",
				term.sequence,
				term.startHeight,
				term.GetEndBlockHeight(),
				term.period,
				term.totalSupply,
				term.totalDelegated,
				len(term.prepSnapshots),
				term.irep,
				term.rrep,
			)
		} else {
			fmt.Fprintf(
				f,
				"Term{%d %d %d %d %s %s %d %s %s}",
				term.sequence,
				term.startHeight,
				term.GetEndBlockHeight(),
				term.period,
				term.totalSupply,
				term.totalDelegated,
				len(term.prepSnapshots),
				term.irep,
				term.rrep,
			)
		}
	case 's':
		fmt.Fprint(f, term.String())
	}
}

func (term *Term) IsDecentralized() bool {
	if term == nil {
		return false
	}
	return term.revision >= icmodule.RevisionDecentralize &&
		len(term.prepSnapshots) > 0 &&
		term.totalDelegated.Sign() == 1
}

func (term *Term) IsFirstBlockOnDecentralized(blockHeight int64) bool {
	return term.IsDecentralized() && term.sequence == 0 && term.startHeight == blockHeight
}

func newTermWithTag(_ icobject.Tag) *Term {
	return &Term{}
}

func newTerm(startHeight, termPeriod int64) *Term {
	return &Term{
		startHeight:    startHeight,
		period:         termPeriod,
		irep:           new(big.Int),
		rrep:           new(big.Int),
		totalSupply:    new(big.Int),
		totalDelegated: new(big.Int),
		rewardFund:     NewRewardFund(),
		prepSnapshots:  nil,

		flags:       FlagNone,
		snapshotMap: nil,
	}
}
