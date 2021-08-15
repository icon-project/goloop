package icstate

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

type PRepSnapshot struct {
	owner            *common.Address
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
	return map[string]interface{}{
		"address":          pss.owner,
		"bondedDelegation": pss.bondedDelegation,
		"delegated":        pss.bondedDelegation,
	}
}

func (pss *PRepSnapshot) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(pss.owner, pss.bondedDelegation)
}

func (pss *PRepSnapshot) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(&pss.owner, &pss.bondedDelegation)
}

func NewPRepSnapshot(owner module.Address, bondedDelegation *big.Int) *PRepSnapshot {
	return &PRepSnapshot{
		owner:            common.AddressToPtr(owner),
		bondedDelegation: bondedDelegation,
	}
}

// =============================================================================

type PRepSnapshots []*PRepSnapshot

func (p PRepSnapshots) Equal(other PRepSnapshots) bool {

	if p == nil && other == nil {
		return true
	}
	if p == nil || other == nil {
		return false
	}
	if len(p) != len(other) {
		return false
	}
	for i, pss := range p {
		if !pss.Equal(other[i]) {
			return false
		}
	}
	return true
}

func (p PRepSnapshots) Clone() PRepSnapshots {
	if p == nil {
		return nil
	}
	ret := make(PRepSnapshots, len(p))
	copy(ret, p)
	return ret
}

func (p PRepSnapshots) toJSON(state *State) []interface{} {
	jso := make([]interface{}, 0, len(p))
	for _, pss := range p {
		ps, _ := state.GetPRepStatusByOwner(pss.Owner(), false)
		grade := ps.Grade()
		if ps != nil && (grade == GradeMain || grade == GradeSub) {
			pssJson := pss.ToJSON()
			pssJson["delegated"] = ps.Delegated()
			jso = append(jso, pssJson)
		}
	}
	return jso
}

// =============================================================================

type termData struct {
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
	// If the length of prepSnapshots is 0, prepSnapshots should be nil
	prepSnapshots PRepSnapshots
}

func (term *termData) Sequence() int {
	return term.sequence
}

func (term *termData) StartHeight() int64 {
	return term.startHeight
}

func (term *termData) Period() int64 {
	return term.period
}

func (term *termData) Irep() *big.Int {
	return term.irep
}

func (term *termData) Rrep() *big.Int {
	return term.rrep
}

func (term *termData) MainPRepCount() int {
	return term.mainPRepCount
}

func (term *termData) GetElectedPRepCount() int {
	return len(term.prepSnapshots)
}

func (term *termData) RewardFund() *RewardFund {
	return term.rewardFund
}

func (term *termData) Iglobal() *big.Int {
	return term.rewardFund.Iglobal
}

func (term *termData) Iprep() *big.Int {
	return term.rewardFund.Iprep
}

func (term *termData) Icps() *big.Int {
	return term.rewardFund.Icps
}

func (term *termData) Irelay() *big.Int {
	return term.rewardFund.Irelay
}

func (term *termData) Ivoter() *big.Int {
	return term.rewardFund.Ivoter
}

func (term *termData) BondRequirement() int {
	return term.bondRequirement
}

func (term *termData) Revision() int {
	return term.revision
}

func (term *termData) GetEndHeight() int64 {
	if term == nil {
		return -1
	}
	return term.startHeight + term.period - 1
}

func (term *termData) GetIISSVersion() int {
	if term.revision >= icmodule.RevisionICON2 {
		return IISSVersion3
	}
	if term.revision >= icmodule.RevisionIISS {
		return IISSVersion2
	}
	return IISSVersion2
}

const DecentralizedHeight = 10362083

func (term *termData) GetVoteStartHeight() int64 {
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

func (term *termData) GetPRepSnapshotCount() int {
	return len(term.prepSnapshots)
}

func (term *termData) GetPRepSnapshotByIndex(index int) *PRepSnapshot {
	return term.prepSnapshots[index]
}

func (term *termData) TotalSupply() *big.Int {
	return term.totalSupply
}

func (term *termData) IsDecentralized() bool {
	return term.isDecentralized
}

func (term *termData) IsFirstBlockOnDecentralized(blockHeight int64) bool {
	return term.isDecentralized && term.sequence == 0 && term.startHeight == blockHeight
}

func (term *termData) equal(other *termData) bool {
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

func (term *termData) clone() termData {
	return termData{
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

func (term *termData) ToJSON(state *State) map[string]interface{} {
	return map[string]interface{}{
		"sequence":              term.sequence,
		"startBlockHeight":      term.startHeight,
		"endBlockHeight":        term.GetEndHeight(),
		"totalSupply":           term.totalSupply,
		"totalDelegated":        term.totalDelegated,
		"totalBondedDelegation": term.getTotalBondedDelegation(),
		"irep":                  term.irep,
		"rrep":                  term.rrep,
		"period":                term.period,
		"rewardFund":            term.rewardFund.ToJSON(),
		"bondRequirement":       term.bondRequirement,
		"revision":              term.revision,
		"isDecentralized":       term.isDecentralized,
		"mainPRepCount":         term.mainPRepCount,
		"iissVersion":           term.GetIISSVersion(),
		"preps":                 term.prepSnapshots.toJSON(state),
	}
}

func (term *termData) getTotalBondedDelegation() *big.Int {
	tbd := new(big.Int)
	for _, snapshot := range term.prepSnapshots {
		tbd.Add(tbd, snapshot.BondedDelegation())
	}
	return tbd
}

func (term *termData) String() string {
	return fmt.Sprintf(
		"Term{seq:%d start:%d end:%d period:%d ts:%s td:%s pss:%d irep:%s rrep:%s revision:%d isDecentralized:%v}",
		term.sequence,
		term.startHeight,
		term.GetEndHeight(),
		term.period,
		term.totalSupply,
		term.totalDelegated,
		len(term.prepSnapshots),
		term.irep,
		term.rrep,
		term.revision,
		term.isDecentralized,
	)
}

func (term *termData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		var format string
		if f.Flag('+') {
			format = "Term{seq:%d start:%d end:%d period:%d totalSupply:%s totalDelegated:%s " +
				"prepSnapshots:%d irep:%s rrep:%s revision:%d isDecentralized:%v}"
		} else {
			format = "Term{%d %d %d %d %s %s %d %s %s %d %v}"
		}
		_, _ = fmt.Fprintf(
			f,
			format,
			term.sequence,
			term.startHeight,
			term.GetEndHeight(),
			term.period,
			term.totalSupply,
			term.totalDelegated,
			len(term.prepSnapshots),
			term.irep,
			term.rrep,
			term.revision,
			term.isDecentralized,
		)
	case 's':
		_, _ = fmt.Fprint(f, term.String())
	}
}

// ========================================================

type TermSnapshot struct {
	icobject.NoDatabase
	termData
}

func (term *TermSnapshot) Version() int {
	return 0
}

func (term *TermSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	return decoder.DecodeAll(
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

func (term *TermSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
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

func (term *TermSnapshot) Equal(o icobject.Impl) bool {
	other, ok := o.(*TermSnapshot)
	if !ok {
		return false
	}
	if term == other {
		return true
	}
	return term.equal(&other.termData)
}

func (term *TermSnapshot) GetPRepSnapshotCount() int {
	return len(term.prepSnapshots)
}

func NewTermWithTag(_ icobject.Tag) *TermSnapshot {
	return &TermSnapshot{}
}

// ==================================================================

type TermState struct {
	snapshot *TermSnapshot
	termData
}

func (term *TermState) GetSnapshot() *TermSnapshot {
	if term.snapshot == nil {
		term.snapshot = &TermSnapshot{
			termData: term.termData.clone(),
		}
	}
	return term.snapshot
}

func (term *TermState) ResetSequence() {
	term.sequence = 0
}

func (term *TermState) SetIsDecentralized(value bool) {
	term.isDecentralized = value
}

func (term *TermState) SetPRepSnapshots(prepSnapshots PRepSnapshots) {
	term.prepSnapshots = prepSnapshots.Clone()
}

func (term *TermState) SetMainPRepCount(mainPRepCount int) {
	term.mainPRepCount = mainPRepCount
}

func (term *TermState) SetIrep(irep *big.Int) {
	term.irep = irep
}

func (term *TermState) SetRrep(rrep *big.Int) {
	term.rrep = rrep
}

func NewNextTerm(state *State, totalSupply *big.Int, revision int) *TermState {
	ts := state.GetTermSnapshot()
	if ts == nil {
		return nil
	}

	return &TermState{
		termData: termData{
			sequence:        ts.Sequence() + 1,
			startHeight:     ts.GetEndHeight() + 1,
			period:          state.GetTermPeriod(),
			irep:            state.GetIRep(),
			rrep:            state.GetRRep(),
			totalSupply:     totalSupply,
			totalDelegated:  state.GetTotalDelegation(),
			rewardFund:      state.GetRewardFund().Clone(),
			bondRequirement: int(state.GetBondRequirement()),
			revision:        revision,
			prepSnapshots:   ts.prepSnapshots.Clone(),
			isDecentralized: ts.IsDecentralized(),
		},
	}
}

func GenesisTerm(state *State, startHeight int64, revision int) *TermState {
	return &TermState{
		termData: termData{
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
		},
	}
}
