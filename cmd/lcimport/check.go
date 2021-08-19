/*
 * Copyright 2021 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/trie"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/icon/iiss/icutils"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type ICON1AccountInfo struct {
	BlockHeight int64              `json:"blockHeight"`
	TermHeight  int64              `json:"termHeight,omitempty"`
	Status      status             `json:"status"`
	Issue       issue              `json:"issue"`
	Front       stage              `json:"front"`
	Accounts    map[string]account `json:"accounts"`
}

func (i *ICON1AccountInfo) Summary() string {
	return fmt.Sprintf("Block height: %d\n"+
		"Term height: %d\n"+
		"Status: %s\n"+
		"Issue: %s\n"+
		"Front: %s\n"+
		"Accounts count: %d\n",
		i.BlockHeight,
		i.TermHeight,
		i.Status.Summary(),
		i.Issue.Summary(),
		i.Front.Summary(),
		len(i.Accounts),
	)
}

type status struct {
	TotalSupply *big.Int `json:"totalSupply"`
	TotalStake  *big.Int `json:"totalStake"`
}

func (s *status) Summary() string {
	return fmt.Sprintf("Supply=%d Stake=%d", s.TotalSupply, s.TotalStake)
}

func (s *status) Check(wss state.WorldSnapshot, extState containerdb.ObjectStoreState) error {
	printTitle("Checking Status")
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	ts := tsVar.BigInt()
	failure := 0
	if s.TotalSupply.Cmp(ts) != 0 {
		fmt.Printf("TotalSupply: icon1(%d) icon2(%d) diff=%d\n",
			s.TotalSupply, ts, new(big.Int).Sub(s.TotalSupply, ts))
		failure += 1
	}

	tsVar = containerdb.NewVarDB(extState,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix, icstate.VarTotalStake))
	ts = tsVar.BigInt()
	if s.TotalStake.Cmp(ts) != 0 {
		fmt.Printf("TotalStake: icon1(%d) icon2(%d) diff=%d\n",
			s.TotalStake, ts, new(big.Int).Sub(s.TotalStake, ts))
		failure += 1
	}
	if failure > 0 {
		return errors.ErrInvalidState
	} else {
		printResult("passed")
		return nil
	}
}

type issue struct {
	IssuedICX        *big.Int `json:"issuedICX"`
	PrevIssuedICX    *big.Int `json:"prevIssuedICX"`
	OverIssuedIScore *big.Int `json:"overIssuedIScore"`
}

func (i *issue) Summary() string {
	return fmt.Sprintf("Issued ICX=%d Prev Issued ICX=%d OverIssued IScore=%d",
		i.IssuedICX, i.PrevIssuedICX, i.OverIssuedIScore)
}

func (i *issue) Check(extState containerdb.ObjectStoreState) error {
	if i == nil {
		return nil
	}
	printTitle("Checking Issue")
	value, err := extState.Get(icstate.IssueKey)
	if err != nil || value == nil {
		return err
	}
	is := icstate.ToIssue(value)
	if is == nil || i.IssuedICX.Cmp(is.TotalReward()) != 0 ||
		i.PrevIssuedICX.Cmp(is.PrevTotalReward()) != 0 ||
		i.OverIssuedIScore.Cmp(is.OverIssuedIScore()) != 0 {
		fmt.Printf("Failed Issue: icon1(%+v) icon2(%+v)\n", i, is)
		return common.ErrInvalidState
	}
	printResult("passed")
	return nil
}

type stage struct {
	Event []event `json:"event"`
}

func (s *stage) Summary() string {
	return fmt.Sprintf("Event count=%d",len(s.Event))
}

const (
	typeDelegation = iota
	typePRepRegister
	typePRepUnregister
)

func (s *stage) Check(stateTerm, front containerdb.ObjectStoreState) error {
	var err error
	if s == nil {
		return nil
	}
	printTitle("Checking Front")
	obj, err := front.Get(icstage.GlobalKey)
	if err != nil || obj == nil {
		return err
	}
	global := icstage.ToGlobal(obj)
	startHeight := global.GetStartHeight()

	acctDB := getAccountDB(stateTerm)
	dMap := make(map[string]icreward.Delegating)
	bh := int64(0)
	index := 0
	failCount := 0
	for i, e := range s.Event {
		offset := e.Height - startHeight
		if bh == e.Height {
			index++
		} else {
			index = 0
			bh = e.Height
		}
		obj = getEventObject(int(offset), index, front)
		if obj == nil {
			fmt.Printf("Failed icstage event %d: icon1(%+v) icon2(nil)\n", i, e)
			failCount++
			continue
		}
		switch e.Type {
		case typeDelegation:
			// update delegating map with Delegation Event
			key := icutils.ToKey(e.Address)
			delegating, ok := dMap[key]
			if !ok {
				// read delegations from state of term start block
				value := acctDB.Get(e.Address)
				if value == nil {
					delegating = icreward.Delegating{}
				} else {
					ea := icstate.ToAccount(value.Object())
					delegating = icreward.Delegating{
						Delegations: ea.Delegations().Clone(),
					}
				}
			}
			voteList := getDelegationEventVoteList(obj)
			if voteList == nil {
				failCount++
				break
			}
			err = delegating.ApplyVotes(voteList)
			if err != nil {
				fmt.Printf("Failed to apply votes %+v\n%+v\n%+v\n", e, delegating, voteList)
				return err
			}
			dMap[key] = delegating

			// compare icon1
			if !compareDelegations(e.Data, delegating.Delegations) {
				fmt.Printf("Failed icstage event %d: icon1(%+v) icon2(%d:%d:%+v) delegating(%+v)\n",
					i, e, offset, index, obj, delegating)
				failCount++
			}
		case typePRepRegister:
			o := icstage.ToEventEnable(obj)
			if !o.Status().IsEnabled() {
				fmt.Printf("Failed icstage event %d: icon1(%+v) icon2(%d:%d:%+v)\n", i, e, offset, index, obj)
				failCount++
			}
		case typePRepUnregister:
			o := icstage.ToEventEnable(obj)
			if o.Status().IsEnabled() {
				fmt.Printf("Failed icstage event %d: icon1(%+v) icon2(%d:%d:%+v)\\n", i, e, offset, index, obj)
				failCount++
			}
		}
	}
	printResult("%d/%d entries got diff values", failCount, len(s.Event))
	if failCount > 0 {
		return common.ErrInvalidState
	}
	return nil
}

func getEventObject(offset, index int, front containerdb.ObjectStoreState) trie.Object {
	key := icstage.EventKey.Append(offset, index).Build()
	obj, err := front.Get(key)
	if err != nil || obj == nil {
		return nil
	}
	return obj
}

func getDelegationEventVoteList(obj trie.Object) icstage.VoteList {
	oType := obj.(*icobject.Object).Tag().Type()
	switch oType {
	case icstage.TypeEventDelegation:
		o := icstage.ToEventVote(obj)
		if o == nil {
			return nil
		}
		return o.Votes()
	case icstage.TypeEventDelegationV2:
		o := icstage.ToEventDelegationV2(obj)
		if o == nil {
			return nil
		}
		return o.Delegating()
	default:
		return nil
	}
}

func compareDelegations(d1, d2 icstate.Delegations) bool {
	if len(d1) != len(d2) {
		return false
	}
	d1Map := d1.ToMap()
	d2Map := d2.ToMap()
	for key, value := range d1Map {
		if !value.Equal(d2Map[key]) {
			return false
		}
	}
	return true
}

type event struct {
	Height  int64               `json:"height"`
	Address *common.Address     `json:"address"`
	Type    int                 `json:"type"`
	Data    icstate.Delegations `json:"data,omitempty"`
}

type account struct {
	Balance     *big.Int              `json:"balance"`
	Stake       *big.Int              `json:"stake,omitempty"`
	Unstakes    []*icstate.Unstake    `json:"unstakes,omitempty"`
	Delegations []*icstate.Delegation `json:"delegations,omitempty"`
	IScore      *big.Int              `json:"iscore,omitempty"`
}

func (a *account) isExtAccountEmpty() bool {
	return (a.Stake == nil || a.Stake.Sign() == 0) &&
		len(a.Unstakes) == 0 &&
		len(a.Delegations) == 0
}

func (a *account) checkBalance(address module.Address, lost *big.Int, wss state.WorldSnapshot) bool {
	var ass state.AccountSnapshot
	ass = wss.GetAccountSnapshot(address.ID())
	if ass == nil {
		if a.Balance.Sign() != 0 {
			fmt.Printf("%s : ICON2 has no world account info\n", address)
			return false
		} else {
			return true
		}
	}
	if address.IsContract() != ass.IsContract() {
		if address.IsContract() {
			fmt.Printf("%s : ICON2 has no contract\n", address)
			return false
		} else {
			lost.Add(lost, a.Balance)
			return true
		}
	}
	if a.Balance.Cmp(ass.GetBalance()) != 0 {
		fmt.Printf("%s: Balance icon1(%d) icon2(%d) diff=%d\n",
			address, a.Balance, ass.GetBalance(), new(big.Int).Sub(a.Balance, ass.GetBalance()))
		return false
	}
	return true
}

func (a *account) checkExtAccount(address module.Address, extState containerdb.ObjectStoreState) bool {
	acctDB := getAccountDB(extState)
	value := acctDB.Get(address)
	if value == nil {
		if !a.isExtAccountEmpty() {
			fmt.Printf("%s : ICON2 has no ext.account info\n", address)
			return false
		} else {
			return true
		}
	}
	ea := icstate.ToAccount(value.Object())

	result := true
	stake := new(big.Int)
	if a.Stake != nil {
		stake = a.Stake
	}
	if stake.Cmp(ea.Stake()) != 0 {
		fmt.Printf("%s: Stake icon1(%d) icon2(%d) diff=%d\n",
			address, stake, ea.Stake(), new(big.Int).Sub(stake, ea.Stake()))
		result = false
	}

	if !ea.UnStakes().Equal(a.Unstakes) {
		fmt.Printf("%s: Unstake icon1(%+v) icon2(%+v)\n",
			address, a.Unstakes, ea.UnStakes())
		result = false
	}

	if !ea.Delegations().Equal(a.Delegations) {
		fmt.Printf("%s: Delegation icon1(%+v) icon2(%+v)\n",
			address, a.Delegations, ea.Delegations())
		result = false
	}

	return result
}

func (a *account) checkIScore(address module.Address, extStages []containerdb.ObjectStoreState, extReward containerdb.ObjectStoreState) bool {
	// get iScore from icreward
	iScore := getIScoreFromReward(address, extReward)

	// get claim data from icstage Front, Back, Back2
	for _, stage := range extStages {
		claim := getIScoreFromStage(address, stage)
		iScore.Sub(iScore, claim)
	}

	icon1 := new(big.Int)
	if a.IScore != nil {
		icon1 = a.IScore
	}
	if icon1.Cmp(iScore) != 0 {
		fmt.Printf("%s: IScore icon1(%d) icon2(%d) diff=%d\n",
			address, icon1, iScore, new(big.Int).Sub(icon1, iScore))
		return false
	}
	return true
}

func getIScoreFromReward(addr module.Address, rewardState containerdb.ObjectStoreState) *big.Int {
	key := icreward.IScoreKey.Append(addr)
	value := containerdb.NewVarDB(rewardState, key)
	is := icreward.ToIScore(value.Object())
	if is == nil {
		return new(big.Int)
	} else {
		return is.Value()
	}
}

func getIScoreFromStage(addr module.Address, stage containerdb.ObjectStoreState) *big.Int {
	key := icstage.IScoreClaimKey.Append(addr)
	value := containerdb.NewVarDB(stage, key)
	is := icstage.ToIScoreClaim(value.Object())
	if is == nil {
		return new(big.Int)
	} else {
		return is.Value()
	}
}

func LoadICON1AccountInfo(path string) (*ICON1AccountInfo, error) {
	printTitle("Load ICON1 export file %s", path)
	jf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jf.Close()

	jd := json.NewDecoder(jf)
	var accountInfo *ICON1AccountInfo
	if err := jd.Decode(&accountInfo); err != nil {
		return nil, err
	}
	fmt.Printf("%s", accountInfo.Summary())
	return accountInfo, nil
}

func CheckState(icon1 *ICON1AccountInfo, wss state.WorldSnapshot, wssTerm state.WorldSnapshot,
	address string, noBalance bool) error {
	height := icon1.BlockHeight
	accounts := icon1.Accounts
	addrSpecified := false
	if len(address) != 0 {
		value, ok := icon1.Accounts[address]
		if ok {
			accounts = make(map[string]account)
			accounts[address] = value
			addrSpecified = true
		} else {
			fmt.Printf("There is no account %s", address)
			return nil
		}
	}
	ess := wss.GetExtensionSnapshot()
	var hashes [][]byte
	if _, err := codec.BC.UnmarshalFromBytes(ess.Bytes(), &hashes); err != nil {
		return err
	}

	extState := getObjectStoreState(wss.Database(), hashes[0], icstate.NewObjectImpl)
	extFront := getObjectStoreState(wss.Database(), hashes[1], icstage.NewObjectImpl)
	extBack1 := getObjectStoreState(wss.Database(), hashes[2], icstage.NewObjectImpl)
	extBack2 := getObjectStoreState(wss.Database(), hashes[3], icstage.NewObjectImpl)
	extStages := []containerdb.ObjectStoreState{extFront, extBack1, extBack2}
	extReward := getObjectStoreState(wss.Database(), hashes[4], icreward.NewObjectImpl)

	iissFailed := 0
	if err := icon1.Status.Check(wss, extState); err != nil {
		iissFailed += 1
	}
	if err := icon1.Issue.Check(extState); err != nil {
		iissFailed += 1
	}
	if icon1.TermHeight != 0 {
		essTerm := wssTerm.GetExtensionSnapshot()
		if _, err := codec.BC.UnmarshalFromBytes(essTerm.Bytes(), &hashes); err != nil {
			return err
		}
		extStateTerm := getObjectStoreState(wssTerm.Database(), hashes[0], icstate.NewObjectImpl)

		if err := icon1.Front.Check(extStateTerm, extFront); err != nil {
			iissFailed += 1
		}
	}
	if noBalance {
		if iissFailed > 0 {
			fmt.Printf("%d different state values @ %d\n", iissFailed, height)
			return errors.InvalidStateError.New("IISSstateComparisonFailure")
		} else {
			return nil
		}
	}
	printTitle("Checking Accounts. %d account entries", len(accounts))
	count := 0
	lost := new(big.Int)
	for key, value := range accounts {
		failed := false
		addr := common.MustNewAddressFromString(key)
		if !value.checkBalance(addr, lost, wss) {
			failed = true
		}
		if !value.checkExtAccount(addr, extState) {
			failed = true
		}
		if !value.checkIScore(addr, extStages, extReward) {
			failed = true
		}
		if failed {
			count++
		}
	}
	if !addrSpecified {
		la := wss.GetAccountSnapshot(state.LostID)
		if lost2 := la.GetBalance(); lost2.Cmp(lost) != 0 {
			fmt.Printf("%s exp=%d real=%d diff=%d\n",
				state.LostAddress, lost, lost2, new(big.Int).Sub(lost2, lost))
		}
	}
	printResult("%d/%d entries got diff values", count, len(accounts))
	if count > 0 || iissFailed > 0 {
		return errors.InvalidStateError.New("FailInComparison")
	}
	return nil
}

func getAccountDB(extState containerdb.ObjectStoreState) *containerdb.DictDB {
	return containerdb.NewDictDB(extState, 1, icstate.AccountDictPrefix)
}

func getObjectStoreSnapshot(dbase db.Database, hash []byte, factory icobject.ImplFactory) *icobject.ObjectStoreSnapshot {
	dbase = icobject.AttachObjectFactory(dbase, factory)
	snapshot := trie_manager.NewImmutableForObject(dbase, hash, icobject.ObjectType)
	return icobject.NewObjectStoreSnapshot(snapshot)
}

func getObjectStoreState(dbase db.Database, hash []byte, factory icobject.ImplFactory) *icobject.ObjectStoreState {
	dbase = icobject.AttachObjectFactory(dbase, factory)
	state := trie_manager.NewMutableForObject(dbase, hash, icobject.ObjectType)
	return icobject.NewObjectStoreState(state)
}

func printTitle(format string, a ...interface{}) {
	fmt.Printf("<<<<< ")
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}

func printResult(format string, a ...interface{}) {
	fmt.Printf("  >> ")
	fmt.Printf(format, a...)
	fmt.Printf("\n")
}