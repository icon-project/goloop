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
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type ICON1AccountInfo struct {
	BlockHeight int64              `json:"blockHeight"`
	Status      status             `json:"status"`
	Accounts    map[string]account `json:"accounts"`
}

type status struct {
	TotalSupply *big.Int	`json:"totalSupply"`
	TotalStake  *big.Int	`json:"totalStake"`
}

func (s *status) Check(wss state.WorldSnapshot, extState containerdb.ObjectStoreState) {
	ass := wss.GetAccountSnapshot(state.SystemID)
	as := scoredb.NewStateStoreWith(ass)
	tsVar := scoredb.NewVarDB(as, state.VarTotalSupply)
	ts := tsVar.BigInt()
	if s.TotalSupply.Cmp(ts) != 0 {
		fmt.Printf("TotalSupply: icon1(%d) icon2(%d) diff=%d\n",
			s.TotalSupply, ts, new(big.Int).Sub(s.TotalSupply, ts))
	}

	tsVar = containerdb.NewVarDB(extState,
		containerdb.ToKey(containerdb.HashBuilder, scoredb.VarDBPrefix, icstate.VarTotalStake))
	ts = tsVar.BigInt()
	if s.TotalStake.Cmp(ts) != 0 {
		fmt.Printf("TotalStake: icon1(%d) icon2(%d) diff=%d\n",
			s.TotalStake, ts, new(big.Int).Sub(s.TotalStake, ts))
	}
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
	return accountInfo, nil
}

func CheckState(icon1 *ICON1AccountInfo, wss state.WorldSnapshot) error {
	height := icon1.BlockHeight
	fmt.Printf("Check %d entries @ %d\n", len(icon1.Accounts), height)
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

	icon1.Status.Check(wss, extState)

	count := 0
	lost := new(big.Int)
	for key, value := range icon1.Accounts {
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
	la := wss.GetAccountSnapshot(state.LostID)
	if lost2 := la.GetBalance(); lost2.Cmp(lost) != 0 {
		fmt.Printf("%s exp=%d real=%d diff=%d\n",
			state.LostAddress, lost, lost2, new(big.Int).Sub(lost2, lost))
	}
	fmt.Printf("%d/%d entries got diff values @ %d\n", count, len(icon1.Accounts), height)
	if count>0 {
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
