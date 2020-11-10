/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iiss

import (
	"math/big"
	"reflect"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

const (
	accountVersion1 = iota + 1
	accountVersion  = accountVersion1

	maxUnstake = 1000
)

var maxUnstakeCount = maxUnstake

func getMaxUnstakeCount() int {
	return maxUnstakeCount
}

func setMaxUnstakeCount(v int) {
	if v == 0 {
		maxUnstakeCount = maxUnstake
	} else {
		maxUnstakeCount = v
	}
}

type Account struct {
	version     int
	staked      *big.Int
	unstakes    unstakeList
	delegated   *big.Int
	delegations delegationList
	bonds       bondList
	unbondings  unbondingList
}

func (a *Account) Version() int {
	return a.version
}

func (a *Account) SetStake(v *big.Int) error {
	if v.Sign() == -1 {
		return errors.Errorf("negative stake is not allowed")
	}
	a.staked = v

	return nil
}

func (a Account) GetStake() *big.Int {
	if a.staked == nil {
		return new(big.Int)
	}
	return a.staked
}

func (a Account) GetUnstakeAmount() *big.Int {
	return a.unstakes.getUnstakeAmount()
}

func (a *Account) UpdateUnstake(stakeInc *big.Int, expireHeight int64) error {
	switch stakeInc.Sign() {
	case 1:
		return a.unstakes.decreaseUnstake(stakeInc)
	case -1:
		return a.unstakes.increaseUnstake(new(big.Int).Abs(stakeInc), expireHeight)
	default:
		return errors.Errorf("Invalid unstake value")
	}
}

// TODO remove unstake via timer

func (a *Account) GetStakeInfo() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	data["stake"] = a.GetStake()
	if unstakes, err := a.unstakes.ToJSON(module.JSONVersion3); err != nil {
		return nil, errors.Errorf("Failed to get unstakes Info")
	} else if unstakes != nil {
		data["unstakes"] = unstakes
	}
	return data, nil
}

func (a *Account) Bytes() []byte {
	if bs, err := codec.BC.MarshalToBytes(a); err != nil {
		panic(err)
	} else {
		return bs
	}
}

func (a *Account) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, a)
	return err
}

func (a *Account) Equal(a2 *Account) bool {
	if a.version != a2.version {
		return false
	}

	if (a.staked == nil && a2.staked == nil) || a.staked.Cmp(a2.staked) != 0 {
		return false
	}

	if a.unstakes.Equal(a2.unstakes) != true {
		return false
	}

	if (a.delegated == nil && a2.delegated == nil) || a.delegated.Cmp(a2.delegated) != 0 {
		return false
	}

	if a.delegations.Equal(a2.delegations) != true {
		return false
	}

	if a.bonds.Equal(a2.bonds) != true {
		return false
	}

	if a.unbondings.Equal(a2.unbondings) != true {
		return false
	}
	return true
}

func (a *Account) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		a.version,
		a.staked,
		a.unstakes,
		a.delegated,
		a.delegations,
		a.bonds,
		a.unbondings,
	); err != nil {
		return err
	}
	return nil
}

func (a *Account) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(
		&a.version,
		&a.staked,
		&a.unstakes,
		&a.delegated,
		&a.delegations,
		&a.bonds,
		&a.unbondings,
	); err != nil {
		return errors.Wrap(err, "Fail to decode account")
	}
	return nil
}

func NewAccount() *Account {
	return &Account{version: accountVersion}
}

type unstake struct {
	amount       *big.Int
	expireHeight int64
}

func (u unstake) equal(u2 unstake) bool {
	return u.amount.Cmp(u2.amount) == 0 && u.expireHeight == u2.expireHeight
}

func (u *unstake) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		u.amount,
		u.expireHeight,
	); err != nil {
		return err
	}
	return nil
}

func (u *unstake) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(
		&u.amount,
		&u.expireHeight,
	); err != nil {
		return errors.Wrap(err, "Fail to decode unstake")
	}
	return nil
}

func (u unstake) ToJSON(v module.JSONVersion) interface{} {
	jso := make(map[string]interface{})

	jso["unstake"] = u.amount
	jso["expireBlockHeight"] = u.expireHeight

	return jso
}

type unstakeList []unstake

func (ul unstakeList) Has() bool {
	return len(ul) > 0
}

func (ul unstakeList) Equal(ul2 unstakeList) bool {
	return reflect.DeepEqual([]unstake(ul), []unstake(ul2))
}

func (ul unstakeList) Clone() unstakeList {
	if ul == nil {
		return nil
	}
	unstakes := make([]unstake, len(ul))
	copy(unstakes, ul)
	return unstakes
}

func (ul unstakeList) getUnstakeAmount() *big.Int {
	total := new(big.Int)
	for _, u := range ul {
		total.Add(total, u.amount)
	}
	return total
}

func (ul *unstakeList) increaseUnstake(v *big.Int, eh int64) error {
	if v.Sign() == -1 {
		return errors.Errorf("Invalid unstake value %v", v)
	}
	av := new(big.Int).Abs(v)
	if len(*ul) >= getMaxUnstakeCount() {
		// update last entry
		lastIndex := len(*ul) - 1
		last := &(*ul)[lastIndex]
		last.amount.Add(last.amount, av)
		if eh > last.expireHeight {
			last.expireHeight = eh
		}
		// TODO update unstake timer
	} else {
		unstake := unstake{
			amount:       av,
			expireHeight: eh,
		}
		// TODO add unstake timer
		*ul = append(*ul, unstake)
	}

	return nil
}

func (ul *unstakeList) decreaseUnstake(v *big.Int) error {
	if v.Sign() == -1 {
		return errors.Errorf("Invalid unstake value %v", v)
	}
	remain := new(big.Int).Set(v)
	unstakes := *ul
	uLen := len(unstakes)
	for i := uLen - 1; i >= 0; i-- {
		us := unstakes[i]
		switch remain.Cmp(us.amount) {
		case 0:
			copy(unstakes[i:], unstakes[i+1:])
			unstakes = unstakes[0 : len(unstakes)-1]
			if len(unstakes) > 0 {
				*ul = unstakes
			} else {
				*ul = nil
			}
			// TODO remove unstake timer
			return nil
		case 1:
			copy(unstakes[i:], unstakes[i+1:])
			unstakes = unstakes[0 : len(unstakes)-1]
			if len(unstakes) > 0 {
				*ul = unstakes
			} else {
				*ul = nil
			}
			// TODO remove unstake timer
			remain.Sub(remain, us.amount)
		case -1:
			us.amount.Sub(us.amount, remain)
			return nil
		}
	}
	return nil
}

func (ul unstakeList) ToJSON(v module.JSONVersion) ([]interface{}, error) {
	if ul.Has() == false {
		return nil, nil
	}
	unstakes := make([]interface{}, len(ul))

	for idx, p := range ul {
		unstakes[idx] = p.ToJSON(v)
	}
	return unstakes, nil
}

type delegation struct {
	target *common.Address
	amount *big.Int
}

func (dg *delegation) equal(dg2 *delegation) bool {
	return dg.target.Equal(dg2.target) && dg.amount.Cmp(dg2.amount) == 0
}

func (dg *delegation) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		dg.target,
		dg.amount,
	); err != nil {
		return err
	}
	return nil
}

func (dg *delegation) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(
		&dg.target,
		&dg.amount,
	); err != nil {
		return errors.Wrap(err, "Fail to decode delegation")
	}
	return nil
}

func (dg delegation) ToJSON(v module.JSONVersion) interface{} {
	jso := make(map[string]interface{})

	jso["address"] = dg.target
	jso["value"] = dg.amount

	return jso
}

type delegationList []delegation

func (dl delegationList) Has() bool {
	return len(dl) > 0
}

func (dl delegationList) Equal(ul2 delegationList) bool {
	return reflect.DeepEqual([]delegation(dl), []delegation(ul2))
}

func (dl delegationList) Clone() delegationList {
	if dl == nil {
		return nil
	}
	delegations := make([]delegation, len(dl))
	copy(delegations, dl)
	return delegations
}

func (dl delegationList) getDelegationAmount() *big.Int {
	total := new(big.Int)
	for _, d := range dl {
		total.Add(total, d.amount)
	}
	return total
}

func (dl delegationList) ToJSON(v module.JSONVersion) ([]interface{}, error) {
	if len(dl) == 0 {
		return nil, nil
	}
	delegations := make([]interface{}, len(dl))

	for idx, d := range dl {
		delegations[idx] = d.ToJSON(v)
	}
	return delegations, nil
}

type bond struct {
	target *common.Address
	amount *big.Int
}

func (b *bond) equal(b2 *bond) bool {
	return b.target.Equal(b2.target) && b.amount.Cmp(b2.amount) == 0
}

func (b *bond) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		b.target,
		b.amount,
	); err != nil {
		return err
	}
	return nil
}

func (b *bond) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(
		&b.target,
		&b.amount,
	); err != nil {
		return errors.Wrap(err, "Fail to decode bond")
	}
	return nil
}

type bondList []bond

func (bl bondList) Has() bool {
	return len(bl) > 0
}

func (bl bondList) Equal(bl2 bondList) bool {
	return reflect.DeepEqual([]bond(bl), []bond(bl2))
}

func (bl bondList) Clone() bondList {
	if bl == nil {
		return nil
	}
	bonds := make([]bond, len(bl))
	copy(bonds, bl)
	return bonds
}

type unbonding struct {
	target       *common.Address
	amount       *big.Int
	expireHeight int64
}

func (ub *unbonding) equal(ub2 *unbonding) bool {
	return ub.target.Equal(ub2.target) && ub.amount.Cmp(ub2.amount) == 0 && ub.expireHeight == ub2.expireHeight
}

func (ub *unbonding) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		ub.target,
		ub.amount,
		ub.expireHeight,
	); err != nil {
		return err
	}
	return nil
}

func (ub *unbonding) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(
		&ub.target,
		&ub.amount,
		&ub.expireHeight,
	); err != nil {
		return errors.Wrap(err, "Fail to decode unbonding")
	}
	return nil
}

type unbondingList []unbonding

func (ul unbondingList) Has() bool {
	return len(ul) > 0
}

func (ul unbondingList) Equal(ul2 unbondingList) bool {
	return reflect.DeepEqual([]unbonding(ul), []unbonding(ul2))
}

func (ul unbondingList) Clone() unbondingList {
	if ul == nil {
		return nil
	}
	unbondings := make([]unbonding, len(ul))
	copy(unbondings, ul)
	return unbondings
}
