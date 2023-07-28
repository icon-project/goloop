package icstate

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/module"
)

type PRepSnapshot struct {
	owner *common.Address
	power *big.Int
}

func (pss *PRepSnapshot) Owner() module.Address {
	return pss.owner
}

func (pss *PRepSnapshot) BondedDelegation() *big.Int {
	return pss.power
}

func (pss *PRepSnapshot) Power() *big.Int {
	return pss.power
}

func (pss *PRepSnapshot) Equal(other *PRepSnapshot) bool {
	if pss == other {
		return true
	}
	if pss == nil || other == nil {
		return false
	}
	return pss.owner.Equal(other.owner) &&
		pss.power.Cmp(other.power) == 0
}

func (pss *PRepSnapshot) Clone() *PRepSnapshot {
	return &PRepSnapshot{
		owner: pss.owner,
		power: pss.power,
	}
}

func (pss *PRepSnapshot) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"address":   pss.owner,
		"power":     pss.power,
		"delegated": pss.power,
	}
}

func (pss *PRepSnapshot) RLPEncodeSelf(e codec.Encoder) error {
	return e.EncodeListOf(pss.owner, pss.power)
}

func (pss *PRepSnapshot) RLPDecodeSelf(d codec.Decoder) error {
	return d.DecodeListOf(&pss.owner, &pss.power)
}

func (pss *PRepSnapshot) String() string {
	return fmt.Sprintf("[%s, %v]", pss.owner, pss.Power())
}

func NewPRepSnapshot(owner module.Address, power *big.Int) *PRepSnapshot {
	return &PRepSnapshot{
		owner: common.AddressToPtr(owner),
		power: power,
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

func (p PRepSnapshots) String() string {
	data := make([]string, len(p))
	for i := 0; i < len(data); i++ {
		data[i] = p[i].String()
	}
	return strings.Join(data, ", ")
}

// =============================================================================

const (
	termVersion1 = iota
	termVersion2
)

type termData struct {
	version         int
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
	rewardFund2     *RewardFund2
	bondRequirement icmodule.Rate
	mainPRepCount   int
	// If the length of prepSnapshots is 0, prepSnapshots should be nil
	prepSnapshots PRepSnapshots
}

func (term *termData) Version() int {
	return term.version
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

func (term *termData) RewardFund2() *RewardFund2 {
	return term.rewardFund2
}

func (term *termData) Iglobal() *big.Int {
	if term.version == termVersion1 {
		return term.rewardFund.Iglobal
	} else {
		return term.rewardFund2.IGlobal()
	}
}

func (term *termData) Iprep() icmodule.Rate {
	return term.rewardFund.Iprep
}

func (term *termData) Icps() icmodule.Rate {
	return term.rewardFund.Icps
}

func (term *termData) Irelay() icmodule.Rate {
	return term.rewardFund.Irelay
}

func (term *termData) Ivoter() icmodule.Rate {
	return term.rewardFund.Ivoter
}

func (term *termData) BondRequirement() icmodule.Rate {
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
	if term.revision >= icmodule.RevisionIISS4 {
		return IISSVersion4
	} else if term.revision >= icmodule.RevisionEnableIISS3 {
		return IISSVersion3
	}
	return IISSVersion2
}

func (term *termData) GetVoteStartHeight() int64 {
	if term.sequence == 0 {
		// If either of initial validators are not registered as PReps
		// when it's decentralized, system will fail
		return term.startHeight + 1
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

	if term.version != other.version {
		return false
	}

	switch term.version {
	case termVersion1:
		if term.rewardFund.Equal(other.rewardFund) == false {
			return false
		}
	case termVersion2:
		if term.rewardFund2.Equal(other.rewardFund2) == false {
			return false
		}
	}

	return term.sequence == other.sequence &&
		term.startHeight == other.startHeight &&
		term.period == other.period &&
		term.irep.Cmp(other.irep) == 0 &&
		term.rrep.Cmp(other.rrep) == 0 &&
		term.totalSupply.Cmp(other.totalSupply) == 0 &&
		term.totalDelegated.Cmp(other.totalDelegated) == 0 &&
		term.bondRequirement == other.bondRequirement &&
		term.revision == other.revision &&
		term.isDecentralized == other.isDecentralized &&
		term.mainPRepCount == other.mainPRepCount &&
		term.prepSnapshots.Equal(other.prepSnapshots)
}

func (term *termData) clone() termData {
	return termData{
		version:         term.version,
		sequence:        term.sequence,
		startHeight:     term.startHeight,
		period:          term.period,
		irep:            term.irep,
		rrep:            term.rrep,
		totalSupply:     term.totalSupply,
		totalDelegated:  term.totalDelegated,
		rewardFund:      term.rewardFund,
		rewardFund2:     term.rewardFund2,
		bondRequirement: term.bondRequirement,
		revision:        term.revision,
		isDecentralized: term.isDecentralized,
		mainPRepCount:   term.mainPRepCount,
		prepSnapshots:   term.prepSnapshots.Clone(),
	}
}

func (term *termData) ToJSON(blockHeight int64, state *State) map[string]interface{} {
	var rf interface{}
	if term.GetIISSVersion() == IISSVersion3 {
		rf = term.rewardFund.ToJSON()
	} else if term.GetIISSVersion() >= IISSVersion4 {
		rf = term.rewardFund2.ToJSON()
	}
	return map[string]interface{}{
		"sequence":         term.sequence,
		"startBlockHeight": term.startHeight,
		"endBlockHeight":   term.GetEndHeight(),
		"totalSupply":      term.totalSupply,
		"totalDelegated":   term.totalDelegated,
		"totalPower":       term.getTotalPower(),
		"irep":             term.irep,
		"rrep":             term.rrep,
		"period":           term.period,
		"rewardFund":       rf,
		"bondRequirement":  term.bondRequirement.Percent(),
		"revision":         term.revision,
		"isDecentralized":  term.isDecentralized,
		"mainPRepCount":    term.mainPRepCount,
		"iissVersion":      term.GetIISSVersion(),
		"preps":            term.prepsToJSON(blockHeight, state),
	}
}

func (term *termData) prepsToJSON(blockHeight int64, state *State) []interface{} {
	br := state.GetBondRequirement()
	jso := make([]interface{}, 0, len(term.prepSnapshots))
	for _, pss := range term.prepSnapshots {
		prep := state.GetPRepByOwner(pss.Owner())
		if prep == nil {
			continue
		}
		grade := prep.Grade()
		if grade == GradeMain || grade == GradeSub {
			prepInJSON := prep.ToJSON(blockHeight, br, 0)
			jso = append(jso, prepInJSON)
		}
	}
	return jso
}

func (term *termData) getTotalPower() *big.Int {
	totalPower := new(big.Int)
	for _, snapshot := range term.prepSnapshots {
		totalPower.Add(totalPower, snapshot.Power())
	}
	return totalPower
}

func (term *termData) String() string {
	return fmt.Sprintf(
		"Term{ver:%d seq:%d start:%d end:%d period:%d ts:%s td:%s pss:%d irep:%s rrep:%s rf:%s, rf2:%s, revision:%d isDecentralized:%v}",
		term.version,
		term.sequence,
		term.startHeight,
		term.GetEndHeight(),
		term.period,
		term.totalSupply,
		term.totalDelegated,
		len(term.prepSnapshots),
		term.irep,
		term.rrep,
		term.rewardFund,
		term.rewardFund2,
		term.revision,
		term.isDecentralized,
	)
}

func (term *termData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		var format string
		var rff string
		if f.Flag('+') {
			format = "Term{ver:%d seq:%d start:%d end:%d period:%d totalSupply:%s totalDelegated:%s " +
				"prepSnapshots:%d irep:%s rrep:%s rf:%s revision:%d isDecentralized:%v}"
			rff = "%+v"
		} else {
			format = "Term{%d %d %d %d %d %s %s %d %s %s %s %d %v}"
			rff = "%v"
		}
		var rf string
		if term.version == termVersion1 {
			rf = fmt.Sprintf(rff, term.rewardFund)
		} else {
			rf = fmt.Sprintf(rff, term.rewardFund2)
		}
		_, _ = fmt.Fprintf(
			f,
			format,
			term.version,
			term.sequence,
			term.startHeight,
			term.GetEndHeight(),
			term.period,
			term.totalSupply,
			term.totalDelegated,
			len(term.prepSnapshots),
			term.irep,
			term.rrep,
			rf,
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

func (term *TermSnapshot) RLPDecodeFields(decoder codec.Decoder) error {
	var bondRequirement int64
	var err error
	switch term.version {
	case termVersion1:
		err = decoder.DecodeAll(
			&term.sequence,
			&term.startHeight,
			&term.period,
			&term.irep,
			&term.rrep,
			&term.totalSupply,
			&term.totalDelegated,
			&term.rewardFund,
			&bondRequirement,
			&term.revision,
			&term.isDecentralized,
			&term.mainPRepCount,
			&term.prepSnapshots,
		)
	case termVersion2:
		err = decoder.DecodeAll(
			&term.sequence,
			&term.startHeight,
			&term.period,
			&term.irep,
			&term.rrep,
			&term.totalSupply,
			&term.totalDelegated,
			&term.rewardFund2,
			&bondRequirement,
			&term.revision,
			&term.isDecentralized,
			&term.mainPRepCount,
			&term.prepSnapshots,
		)
	}
	if err == nil {
		term.bondRequirement = icmodule.ToRate(bondRequirement)
	}
	return err
}

func (term *TermSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	switch term.version {
	case termVersion1:
		return encoder.EncodeMulti(
			term.sequence,
			term.startHeight,
			term.period,
			term.irep,
			term.rrep,
			term.totalSupply,
			term.totalDelegated,
			term.rewardFund,
			term.bondRequirement.Percent(),
			term.revision,
			term.isDecentralized,
			term.mainPRepCount,
			term.prepSnapshots,
		)
	case termVersion2:
		return encoder.EncodeMulti(
			term.sequence,
			term.startHeight,
			term.period,
			term.irep,
			term.rrep,
			term.totalSupply,
			term.totalDelegated,
			term.rewardFund2,
			term.bondRequirement.Percent(),
			term.revision,
			term.isDecentralized,
			term.mainPRepCount,
			term.prepSnapshots,
		)
	default:
		return errors.IllegalArgumentError.Errorf("illegal Term version %d", term.version)
	}
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

func NewTermWithTag(tag icobject.Tag) *TermSnapshot {
	return &TermSnapshot{
		termData: termData{
			version: tag.Version(),
		},
	}
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
	var version int
	var rf *RewardFund
	var rf2 *RewardFund2
	if revision < icmodule.RevisionIISS4 {
		version = termVersion1
		rf = state.GetRewardFund()
	} else {
		version = termVersion2
		rf2 = state.GetRewardFund2()
	}

	return &TermState{
		termData: termData{
			version:         version,
			sequence:        ts.Sequence() + 1,
			startHeight:     ts.GetEndHeight() + 1,
			period:          state.GetTermPeriod(),
			irep:            state.GetIRep(),
			rrep:            state.GetRRep(),
			totalSupply:     totalSupply,
			totalDelegated:  state.GetTotalDelegation(),
			rewardFund:      rf,
			rewardFund2:     rf2,
			bondRequirement: state.GetBondRequirement(),
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
			bondRequirement: state.GetBondRequirement(),
			revision:        revision,
			isDecentralized: false,
		},
	}
}
