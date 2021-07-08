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
	"io/ioutil"
	"math/big"
	"os"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/trie/trie_manager"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/state"
)

type ICON1AccountInfo struct {
	Block    int64              `json:"block"`
	Accounts map[string]account `json:"accounts"`
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

func (a *account) checkBalance(address module.Address, wss state.WorldSnapshot) bool {
	ass := wss.GetAccountSnapshot(address.ID())
	if ass == nil {
		if a.Balance.Sign() != 0 {
			fmt.Printf("%s : ICON2 has no world account info\n", address)
			return false
		} else {
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

func (a *account) checkExtAccount(address module.Address, dbase db.Database, hash []byte) bool {
	acctDB := getAccountDB(dbase, hash)
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

func (a *account) checkIScore(address module.Address, dbase db.Database, hashes [][]byte) bool {
	// get iScore from icreward
	iScore := getIScoreFromReward(address, dbase, hashes[4])

	// get claim data from icstage Front, Back, Back2
	for i := 1; i < 4; i++ {
		claim := getIScoreFromStage(address, dbase, hashes[i])
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

func getIScoreFromReward(addr module.Address, dbase db.Database, hash []byte) *big.Int {
	iscore := new(big.Int)
	oss := getObjectStoreSnapshot(dbase, hash, icreward.NewObjectImpl)
	key := icreward.IScoreKey.Append(addr)
	value := containerdb.NewVarDB(oss, key)
	if value == nil {
		return iscore
	}
	is := icreward.ToIScore(value.Object())
	if is != nil {
		iscore.Set(is.Value())
	}
	return iscore
}

func getIScoreFromStage(addr module.Address, dbase db.Database, hash []byte) *big.Int {
	oss := getObjectStoreSnapshot(dbase, hash, icstage.NewObjectImpl)
	key := icstage.IScoreClaimKey.Append(addr)
	value := containerdb.NewVarDB(oss, key)
	iscore := new(big.Int)
	if value == nil {
		return iscore
	}
	is := icstage.ToIScoreClaim(value.Object())
	if is != nil {
		iscore.Set(is.Value())
	}
	return iscore
}

func LoadICON1AccountInfo(path string) (*ICON1AccountInfo, error) {
	jf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jf.Close()

	bs, err := ioutil.ReadAll(jf)
	if err != nil {
		return nil, err
	}

	accountInfo := new(ICON1AccountInfo)
	err = json.Unmarshal(bs, accountInfo)
	if err != nil {
		return nil, err
	}

	return accountInfo, nil
}

func CheckState(icon1 *ICON1AccountInfo, wss state.WorldSnapshot) error {
	height := icon1.Block
	fmt.Printf("Check %d entries @ %d\n", len(icon1.Accounts), height)
	ess := wss.GetExtensionSnapshot()
	var hashes [][]byte
	if _, err := codec.BC.UnmarshalFromBytes(ess.Bytes(), &hashes); err != nil {
		return err
	}

	count := 0
	for key, value := range icon1.Accounts {
		failed := false
		addr := common.MustNewAddressFromString(key)
		if !value.checkBalance(addr, wss) {
			failed = true
		}
		if !value.checkExtAccount(addr, wss.Database(), hashes[0]) {
			failed = true
		}
		if !value.checkIScore(addr, wss.Database(), hashes) {
			failed = true
		}
		if failed {
			count++
		}
	}
	fmt.Printf("%d/%d entries got diff values @ %d\n", count, len(icon1.Accounts), height)
	return nil
}

func getAccountDB(dbase db.Database, hash []byte) *containerdb.DictDB {
	oss := getObjectStoreSnapshot(dbase, hash, icstate.NewObjectImpl)
	return containerdb.NewDictDB(oss, 1, icstate.AccountDictPrefix)
}

func getObjectStoreSnapshot(dbase db.Database, hash []byte, factory icobject.ImplFactory) *icobject.ObjectStoreSnapshot {
	dbase = icobject.AttachObjectFactory(dbase, factory)
	snapshot := trie_manager.NewImmutableForObject(dbase, hash, icobject.ObjectType)
	return icobject.NewObjectStoreSnapshot(snapshot)
}
