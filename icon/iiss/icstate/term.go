package icstate

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
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

func NewPRepSnapshot(owner module.Address, bondedDelegation *big.Int) *PRepSnapshot {
	return &PRepSnapshot{
		owner:            owner,
		bondedDelegation: bondedDelegation,
	}
}

type PRepSnapshots struct {
	pssList []*PRepSnapshot

	pssMap                map[string]int
	totalBondedDelegation *big.Int
}

func (p *PRepSnapshots) Equal(other *PRepSnapshots) bool {
	if p == other {
		return true
	}
	if p == nil || other == nil {
		return false
	}
	if p.Len() != other.Len() {
		return false
	}

	for i, pss := range p.pssList {
		if !pss.Equal(other.pssList[i]) {
			return false
		}
	}
	return true
}

func (p *PRepSnapshots) Clone() *PRepSnapshots {
	if p == nil {
		return nil
	}

	size := p.Len()
	pssList := make([]*PRepSnapshot, size, size)
	pssMap := make(map[string]int)

	for i, pss := range p.pssList {
		pssList[i] = pss.Clone()
		pssMap[icutils.ToKey(pss.Owner())] = i
	}
	return &PRepSnapshots{
		pssList:               pssList,
		pssMap:                pssMap,
		totalBondedDelegation: p.totalBondedDelegation,
	}
}

func (p *PRepSnapshots) toJSON() []interface{} {
	size := p.Len()
	jso := make([]interface{}, size, size)

	for i, pss := range p.pssList {
		jso[i] = pss.ToJSON()
	}

	return jso
}

func (p *PRepSnapshots) RLPDecodeSelf(d codec.Decoder) error {
	if err := d.Decode(&p.pssList); err != nil {
		return err
	}

	tbd := new(big.Int)
	p.pssMap = make(map[string]int)

	for i, pss := range p.pssList {
		p.pssMap[icutils.ToKey(pss.Owner())] = i
		tbd.Add(tbd, pss.BondedDelegation())
	}
	p.totalBondedDelegation = tbd
	return nil
}

func (p *PRepSnapshots) RLPEncodeSelf(e codec.Encoder) error {
	return e.Encode(p.pssList)
}

func (p *PRepSnapshots) IndexOf(value interface{}) int {
	owner := value.(module.Address)
	i, ok := p.pssMap[icutils.ToKey(owner)]
	if !ok {
		return -1
	}
	return i
}

func (p *PRepSnapshots) Get(i int) interface{} {
	if i < 0 || i >= p.Len() {
		return nil
	}
	return p.pssList[i]
}

func (p *PRepSnapshots) Len() int {
	if p == nil {
		return 0
	}
	return len(p.pssList)
}

func (p *PRepSnapshots) TotalBondedDelegation() *big.Int {
	return p.totalBondedDelegation
}

func (p *PRepSnapshots) append(i int, owner module.Address, bondedDelegation *big.Int) {
	pss := NewPRepSnapshot(owner, bondedDelegation)
	p.pssList = append(p.pssList, pss)
	p.pssMap[icutils.ToKey(owner)] = i
}

func NewEmptyPRepSnapshots() *PRepSnapshots {
	return &PRepSnapshots{
		pssMap:                make(map[string]int),
		totalBondedDelegation: new(big.Int),
	}
}

func NewPRepSnapshots(preps *PReps, electedPRepCount int, br int64) *PRepSnapshots {
	size := icutils.Min(preps.Size(), electedPRepCount)
	var pssList []*PRepSnapshot = make([]*PRepSnapshot, size, size)
	pssMap := make(map[string]int)
	tbd := new(big.Int)

	for i := 0; i < size; i++ {
		prep := preps.GetPRepByIndex(i)
		owner := prep.Owner()
		pss := NewPRepSnapshot(owner, prep.GetBondedDelegation(br))

		pssList[i] = pss
		pssMap[icutils.ToKey(owner)] = i
		tbd.Add(tbd, pss.BondedDelegation())
	}

	return &PRepSnapshots{
		pssList:               pssList,
		pssMap:                pssMap,
		totalBondedDelegation: tbd,
	}
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
	revision        int
	isDecentralized bool
	irep            *big.Int
	rrep            *big.Int
	totalSupply     *big.Int
	totalDelegated  *big.Int // total delegated amount of all active P-Reps. Set with PRepManager.totalDelegated
	rewardFund      *RewardFund
	bondRequirement int
	mainPRepCount   int
	prepSnapshots   *PRepSnapshots

	flags TermFlag
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
	return term.prepSnapshots.Len()
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

func (term *Term) SetRevision(revision int) {
	term.revision = revision
}

func (term *Term) GetEndHeight() int64 {
	if term == nil {
		return -1
	}
	return term.startHeight + term.period - 1
}

func (term *Term) GetIISSVersion() int {
	if term.revision >= icmodule.RevisionICON2 {
		return IISSVersion3
	}
	if term.revision >= icmodule.RevisionIISS {
		return IISSVersion2
	}
	return IISSVersion2
}

const DecentralizedHeight = 10362083

func (term *Term) GetVoteStartHeight() int64 {
	if term.sequence == 0 {
		if term.startHeight == DecentralizedHeight {
			// It's decentralized in main network under LOOPCHAIN
			return term.startHeight + 1
		} else {
			// It's decentralized in test network under GOLOOP
			return term.startHeight + 2
		}
	}
	return -1
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
	term.isDecentralized = other.isDecentralized
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
		isDecentralized: term.isDecentralized,
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
		&term.isDecentralized,
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
		term.isDecentralized,
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
		term.isDecentralized == other.isDecentralized &&
		term.mainPRepCount == other.mainPRepCount &&
		term.prepSnapshots.Equal(other.prepSnapshots)
}

func (term *Term) GetPRepSnapshotCount() int {
	return term.prepSnapshots.Len()
}

func (term *Term) GetPRepSnapshotByIndex(index int) *PRepSnapshot {
	return term.prepSnapshots.Get(index).(*PRepSnapshot)
}

func (term *Term) GetPRepSnapshotByOwner(owner module.Address) *PRepSnapshot {
	i := term.prepSnapshots.IndexOf(owner)
	if i < 0 {
		return nil
	}
	return term.prepSnapshots.Get(i).(*PRepSnapshot)
}

func (term *Term) getPRepSnapshotIndex(owner module.Address) int {
	return term.prepSnapshots.IndexOf(owner)
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

func (term *Term) IsDecentralized() bool {
	return term.isDecentralized
}

func (term *Term) SetIsDecentralized(value bool) {
	term.isDecentralized = value
}

func (term *Term) TotalBondedDelegation() *big.Int {
	if term.prepSnapshots != nil {
		return term.prepSnapshots.TotalBondedDelegation()
	}
	return new(big.Int)
}

func (term *Term) ToJSON() map[string]interface{} {
	jso := make(map[string]interface{})

	jso["sequence"] = term.sequence
	jso["startBlockHeight"] = term.startHeight
	jso["endBlockHeight"] = term.GetEndHeight()
	jso["totalSupply"] = term.totalSupply
	jso["totalDelegated"] = term.totalDelegated
	jso["totalBondedDelegation"] = term.TotalBondedDelegation()
	jso["irep"] = term.irep
	jso["rrep"] = term.rrep
	jso["period"] = term.period
	jso["rewardFund"] = term.rewardFund.ToJSON()
	jso["bondRequirement"] = term.bondRequirement
	jso["revision"] = term.revision
	jso["isDecentralized"] = term.isDecentralized
	jso["mainPRepCount"] = term.mainPRepCount
	jso["iissVersion"] = term.GetIISSVersion()
	jso["preps"] = term.prepSnapshots.toJSON()

	return jso
}

func NewNextTerm(state *State, totalSupply *big.Int, revision int) *Term {
	term := state.GetTerm()
	if term == nil {
		return nil
	}

	return &Term{
		sequence:        term.sequence + 1,
		startHeight:     term.GetEndHeight() + 1,
		period:          state.GetTermPeriod(),
		irep:            state.GetIRep(),
		rrep:            state.GetRRep(),
		totalSupply:     totalSupply,
		totalDelegated:  state.GetTotalDelegation(),
		rewardFund:      state.GetRewardFund().Clone(),
		bondRequirement: int(state.GetBondRequirement()),
		revision:        revision,
		prepSnapshots:   term.prepSnapshots.Clone(),
		isDecentralized: term.IsDecentralized(),

		flags: term.flags | FlagNextTerm,
	}
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
		isDecentralized: false,

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

func (term *Term) SetPRepSnapshots(prepSnapshots *PRepSnapshots) {
	term.checkWritable()
	term.prepSnapshots = prepSnapshots
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
		"Term: seq=%d start=%d end=%d period=%d ts=%s td=%s pss=%d irep=%s rrep=%s revision=%d isDecentralized=%v",
		term.sequence,
		term.startHeight,
		term.GetEndHeight(),
		term.period,
		term.totalSupply,
		term.totalDelegated,
		term.prepSnapshots.Len(),
		term.irep,
		term.rrep,
		term.revision,
		term.isDecentralized,
	)
}

func (term *Term) Format(f fmt.State, c rune) {
	size := term.prepSnapshots.Len()
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(
				f,
				"Term{seq=%d start=%d end=%d period=%d totalSupply=%s totalDelegated=%s "+
					"prepSnapshot=%d irep=%s rrep=%s revision=%d isDecentralized=%v}",
				term.sequence,
				term.startHeight,
				term.GetEndHeight(),
				term.period,
				term.totalSupply,
				term.totalDelegated,
				size,
				term.irep,
				term.rrep,
				term.revision,
				term.isDecentralized,
			)
		} else {
			fmt.Fprintf(
				f,
				"Term{%d %d %d %d %s %s %d %s %s %d %v}",
				term.sequence,
				term.startHeight,
				term.GetEndHeight(),
				term.period,
				term.totalSupply,
				term.totalDelegated,
				size,
				term.irep,
				term.rrep,
				term.revision,
				term.isDecentralized,
			)
		}
	case 's':
		fmt.Fprint(f, term.String())
	}
}

func (term *Term) IsFirstBlockOnDecentralized(blockHeight int64) bool {
	return term.IsDecentralized() && term.sequence == 0 && term.startHeight == blockHeight
}

func NewTermWithTag(_ icobject.Tag) *Term {
	return &Term{}
}

func NewTerm(startHeight, termPeriod int64) *Term {
	return &Term{
		startHeight:    startHeight,
		period:         termPeriod,
		irep:           new(big.Int),
		rrep:           new(big.Int),
		totalSupply:    new(big.Int),
		totalDelegated: new(big.Int),
		rewardFund:     NewRewardFund(),
		prepSnapshots:  NewEmptyPRepSnapshots(),

		flags: FlagNone,
	}
}
