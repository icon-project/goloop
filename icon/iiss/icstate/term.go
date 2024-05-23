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
	return fmt.Sprintf("PRepSnapshot{%s %d}", pss.owner, pss.Power())
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
	var sb strings.Builder
	sb.WriteString("PRepSnapshots{")
	for i, pss := range p {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(pss.String())
	}
	sb.WriteByte('}')
	return sb.String()
}

// =============================================================================

const (
	termVersion1 = iota
	termVersion2
	termVersionReserved
)

type termDataCommon struct {
	version         int
	sequence        int
	startHeight     int64
	period          int64
	revision        int
	isDecentralized bool
	totalSupply     *big.Int    // not nil
	totalDelegated  *big.Int    // total delegated amount of all active P-Reps. not nil
	rewardFund      *RewardFund // not nil
	bondRequirement icmodule.Rate
	mainPRepCount   int
	// If the length of prepSnapshots is 0, prepSnapshots should be nil
	prepSnapshots PRepSnapshots
}

func (term *termDataCommon) Version() int {
	return term.version
}

func (term *termDataCommon) Sequence() int {
	return term.sequence
}

func (term *termDataCommon) StartHeight() int64 {
	return term.startHeight
}

func (term *termDataCommon) Period() int64 {
	return term.period
}

func (term *termDataCommon) MainPRepCount() int {
	return term.mainPRepCount
}

func (term *termDataCommon) GetElectedPRepCount() int {
	return len(term.prepSnapshots)
}

func (term *termDataCommon) RewardFund() *RewardFund {
	return term.rewardFund
}

func (term *termDataCommon) BondRequirement() icmodule.Rate {
	return term.bondRequirement
}

func (term *termDataCommon) Revision() int {
	return term.revision
}

func (term *termDataCommon) GetEndHeight() int64 {
	if term == nil {
		return -1
	}
	return term.startHeight + term.period - 1
}

func (term *termDataCommon) GetIISSVersion() int {
	if term.revision >= icmodule.RevisionIISS4R1 {
		return IISSVersion4
	} else if term.revision >= icmodule.RevisionEnableIISS3 {
		return IISSVersion3
	}
	return IISSVersion2
}

func (term *termDataCommon) GetVoteStartHeight() int64 {
	if term.sequence == 0 {
		// If either of initial validators are not registered as PReps
		// when it's decentralized, system will fail
		return term.startHeight + 1
	}
	return -1
}

func (term *termDataCommon) PRepSnapshots() PRepSnapshots {
	return term.prepSnapshots
}

func (term *termDataCommon) TotalSupply() *big.Int {
	return term.totalSupply
}

func (term *termDataCommon) IsDecentralized() bool {
	return term.isDecentralized
}

func (term *termDataCommon) equal(other *termDataCommon) bool {
	if term == other {
		return true
	}
	if term == nil || other == nil {
		return false
	}
	return term.version == other.version &&
		term.sequence == other.sequence &&
		term.startHeight == other.startHeight &&
		term.period == other.period &&
		term.totalSupply.Cmp(other.totalSupply) == 0 &&
		term.totalDelegated.Cmp(other.totalDelegated) == 0 &&
		term.bondRequirement == other.bondRequirement &&
		term.revision == other.revision &&
		term.isDecentralized == other.isDecentralized &&
		term.mainPRepCount == other.mainPRepCount &&
		term.rewardFund.Equal(other.rewardFund) &&
		term.prepSnapshots.Equal(other.prepSnapshots)
}

func (term *termDataCommon) clone() termDataCommon {
	return termDataCommon{
		version:         term.version,
		sequence:        term.sequence,
		startHeight:     term.startHeight,
		period:          term.period,
		totalSupply:     term.totalSupply,
		totalDelegated:  term.totalDelegated,
		rewardFund:      term.rewardFund,
		bondRequirement: term.bondRequirement,
		revision:        term.revision,
		isDecentralized: term.isDecentralized,
		mainPRepCount:   term.mainPRepCount,
		prepSnapshots:   term.prepSnapshots.Clone(),
	}
}

func (term *termDataCommon) ToJSON(sc icmodule.StateContext, state *State) map[string]interface{} {
	return map[string]interface{}{
		"sequence":         term.sequence,
		"startBlockHeight": term.startHeight,
		"endBlockHeight":   term.GetEndHeight(),
		"totalSupply":      term.totalSupply,
		"totalDelegated":   term.totalDelegated,
		"totalPower":       term.getTotalPower(),
		"period":           term.period,
		"rewardFund":       term.rewardFund.ToJSON(),
		"bondRequirement":  term.bondRequirement.Percent(),
		"revision":         term.revision,
		"isDecentralized":  term.isDecentralized,
		"mainPRepCount":    term.mainPRepCount,
		"iissVersion":      term.GetIISSVersion(),
		"preps":            term.prepsToJSON(sc, state),
	}
}

func (term *termDataCommon) prepsToJSON(sc icmodule.StateContext, state *State) []interface{} {
	jso := make([]interface{}, 0, len(term.prepSnapshots))
	for _, pss := range term.prepSnapshots {
		prep := state.GetPRepByOwner(pss.Owner())
		if prep == nil {
			continue
		}
		grade := prep.Grade()
		if grade == GradeMain || grade == GradeSub {
			prepInJSON := prep.ToJSON(sc)
			jso = append(jso, prepInJSON)
		}
	}
	return jso
}

func (term *termDataCommon) getTotalPower() *big.Int {
	totalPower := new(big.Int)
	for _, snapshot := range term.prepSnapshots {
		totalPower.Add(totalPower, snapshot.Power())
	}
	return totalPower
}

func (term *termDataCommon) String() string {
	return fmt.Sprintf(
		"ver=%d seq=%d start=%d end=%d period=%d ts=%s td=%s pss=%d rf=%s rev=%d isDec=%t",
		term.version,
		term.sequence,
		term.startHeight,
		term.GetEndHeight(),
		term.period,
		term.totalSupply,
		term.totalDelegated,
		len(term.prepSnapshots),
		term.rewardFund,
		term.revision,
		term.isDecentralized,
	)
}

// ========================================================

type termDataExtV1 struct {
	irep *big.Int // not nil
	rrep *big.Int // not nil
}

func (tde *termDataExtV1) Irep() *big.Int {
	if tde == nil {
		return icmodule.BigIntZero
	}
	return tde.irep
}

func (tde *termDataExtV1) Rrep() *big.Int {
	if tde == nil {
		return icmodule.BigIntZero
	}
	return tde.rrep
}

func (tde *termDataExtV1) equal(other *termDataExtV1) bool {
	if tde == other {
		return true
	}
	if tde == nil || other == nil {
		return false
	}
	return tde.irep.Cmp(other.irep) == 0 && tde.rrep.Cmp(other.rrep) == 0
}

func (tde *termDataExtV1) String() string {
	return fmt.Sprintf("irep=%d rrep=%d", tde.Irep(), tde.Rrep())
}

func (tde *termDataExtV1) clone() *termDataExtV1 {
	if tde == nil {
		return nil
	}
	return newTermDataExtV1(tde.irep, tde.rrep)
}

func newTermDataExtV1(irep, rrep *big.Int) *termDataExtV1 {
	return &termDataExtV1{irep, rrep}
}

// ========================================================

type termDataExtV2 struct {
	minimumBond *big.Int // not nil
}

func (tde *termDataExtV2) MinimumBond() *big.Int {
	if tde == nil {
		return icmodule.BigIntZero
	}
	return tde.minimumBond
}

func (tde *termDataExtV2) equal(other *termDataExtV2) bool {
	if tde == other {
		return true
	}
	if tde == nil || other == nil {
		return false
	}
	return tde.minimumBond.Cmp(other.minimumBond) == 0
}

func (tde *termDataExtV2) String() string {
	return fmt.Sprintf("minBond=%d", tde.MinimumBond())
}

func (tde *termDataExtV2) clone() *termDataExtV2 {
	if tde == nil {
		return nil
	}
	return &termDataExtV2{tde.minimumBond}
}

func newTermDataExtV2(minimumBond *big.Int) *termDataExtV2 {
	return &termDataExtV2{minimumBond}
}

// ========================================================

type termData struct {
	termDataCommon
	*termDataExtV1
	*termDataExtV2
}

func (term *termData) equal(other *termData) bool {
	if !term.termDataCommon.equal(&other.termDataCommon) {
		return false
	}
	switch term.Version() {
	case termVersion1:
		return term.termDataExtV1.equal(other.termDataExtV1)
	case termVersion2:
		return term.termDataExtV2.equal(other.termDataExtV2)
	}
	return true
}

func (term *termData) clone() termData {
	td := termData{
		termDataCommon: term.termDataCommon.clone(),
	}
	switch term.Version() {
	case termVersion1:
		td.termDataExtV1 = term.termDataExtV1.clone()
	case termVersion2:
		td.termDataExtV2 = term.termDataExtV2.clone()
	}
	return td
}

func (term *termData) ToJSON(sc icmodule.StateContext, state *State) map[string]interface{} {
	jso := term.termDataCommon.ToJSON(sc, state)
	switch term.Version() {
	case termVersion1:
		jso["irep"] = term.Irep()
		jso["rrep"] = term.Rrep()
	case termVersion2:
		jso["minimumBond"] = term.MinimumBond()
	}
	return jso
}

func (term *termData) String() string {
	sb := strings.Builder{}
	sb.WriteString("Term{")
	sb.WriteString(term.termDataCommon.String())
	sb.WriteByte(' ')
	switch term.Version() {
	case termVersion1:
		sb.WriteString(term.termDataExtV1.String())
	case termVersion2:
		sb.WriteString(term.termDataExtV2.String())
	}
	sb.WriteByte('}')
	return sb.String()
}

func (term *termData) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		var format string
		if f.Flag('+') {
			format = "Term{ver=%d seq=%d start=%d end=%d period=%d ts=%d td=%d " +
				"pss=%d irep=%s rrep=%s rf=%+v rev=%d isDec=%t minBond=%d}"
		} else {
			format = "Term{%d %d %d %d %d %d %d %d %s %s %v %d %t %d}"
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
			term.Irep(),
			term.Rrep(),
			term.rewardFund,
			term.revision,
			term.isDecentralized,
			term.MinimumBond(),
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
	switch term.Version() {
	case termVersion1:
		if err := decoder.DecodeAll(
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
		); err != nil {
			return err
		}
		term.bondRequirement = icmodule.ToRate(bondRequirement)
	case termVersion2:
		if err := decoder.DecodeAll(
			&term.sequence,
			&term.startHeight,
			&term.period,
			&term.totalSupply,
			&term.totalDelegated,
			&term.rewardFund,
			&bondRequirement,
			&term.revision,
			&term.isDecentralized,
			&term.mainPRepCount,
			&term.prepSnapshots,
			&term.minimumBond,
		); err != nil {
			return err
		}
		term.bondRequirement = icmodule.Rate(bondRequirement)
	}
	return nil
}

func (term *TermSnapshot) RLPEncodeFields(encoder codec.Encoder) error {
	switch term.Version() {
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
			term.totalSupply,
			term.totalDelegated,
			term.rewardFund,
			term.bondRequirement.NumInt64(),
			term.revision,
			term.isDecentralized,
			term.mainPRepCount,
			term.prepSnapshots,
			term.minimumBond,
		)
	default:
		return errors.IllegalArgumentError.Errorf("IllegalTermVersion(%d)", term.Version())
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

func NewTermWithTag(tag icobject.Tag) *TermSnapshot {
	version := tag.Version()
	tss := &TermSnapshot{
		termData: termData{
			termDataCommon: termDataCommon{version: version},
		},
	}
	switch version {
	case termVersion1:
		tss.termDataExtV1 = newTermDataExtV1(icmodule.BigIntZero, icmodule.BigIntZero)
	case termVersion2:
		tss.termDataExtV2 = newTermDataExtV2(icmodule.BigIntZero)
	}
	return tss
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

func (term *TermState) SetIrep(irep *big.Int) {
	if term.termDataExtV1 != nil {
		term.irep = irep
	}
}

func (term *TermState) SetRrep(rrep *big.Int) {
	if term.termDataExtV1 != nil {
		term.rrep = rrep
	}
}

// NewNextTerm returns the next term
// It assumes that state and totalSupply are not nil.
func NewNextTerm(sc icmodule.StateContext, state *State, totalSupply *big.Int, preps PRepSet) *TermState {
	rev := sc.RevisionValue()
	// Previous term
	tss := state.GetTermSnapshot()
	if tss == nil {
		return nil
	}
	var version int
	if rev < icmodule.RevisionIISS4R1 {
		version = termVersion1
	} else {
		version = termVersion2
	}

	isDecentralized := preps != nil
	sequence := tss.Sequence() + 1
	if !tss.IsDecentralized() && isDecentralized {
		// When decentralization begins, reset sequence to 0
		sequence = 0
	}

	// New a TermState instance for the next term
	termState := &TermState{
		termData: termData{
			termDataCommon: termDataCommon{
				version:         version,
				sequence:        sequence,
				startHeight:     tss.GetEndHeight() + 1,
				period:          state.GetTermPeriod(),
				totalSupply:     totalSupply,
				totalDelegated:  state.GetTotalDelegation(),
				rewardFund:      state.GetRewardFund(rev),
				bondRequirement: state.GetBondRequirement(rev),
				revision:        rev,
				isDecentralized: isDecentralized,
			},
		},
	}
	switch version {
	case termVersion1:
		termState.termDataExtV1 = newTermDataExtV1(state.GetIRep(), state.GetRRep())
	case termVersion2:
		termState.termDataExtV2 = newTermDataExtV2(state.GetMinimumBond())
	}

	// Update PRepSnapshots
	if isDecentralized {
		termState.mainPRepCount = preps.GetPRepSize(GradeMain)
		termState.prepSnapshots = preps.ToPRepSnapshots(sc.GetBondRequirement())
	}

	return termState
}

func GenesisTerm(state *State, startHeight int64, revision int) *TermState {
	return &TermState{
		termData: termData{
			termDataCommon: termDataCommon{
				version:         termVersion1,
				sequence:        0,
				startHeight:     startHeight,
				period:          state.GetTermPeriod(),
				totalSupply:     icmodule.BigIntZero,
				totalDelegated:  icmodule.BigIntZero,
				rewardFund:      state.GetRewardFundV1().Clone(),
				bondRequirement: state.GetBondRequirement(revision),
				revision:        revision,
				isDecentralized: false,
			},
			termDataExtV1: newTermDataExtV1(state.GetIRep(), state.GetRRep()),
		},
	}
}
