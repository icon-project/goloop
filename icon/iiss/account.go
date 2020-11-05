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
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/service/state"
)

const (
	accountVersion1 = iota + 1
	accountVersion = accountVersion1

	maxUnstake = 100
)

type AccountState interface {
	Bytes() []byte
	SetBytes(bs []byte) error
	Version() int
	SetStake(account state.AccountState, v *common.HexInt) error
	GetStake() *common.HexInt
}

type accountStateImpl struct {
	version		int
	staked		*common.HexInt
	//delegated	*common.HexInt
	unstakes	[]*unstake
	//delegations []*Delegation
	//bonds		[]*Bond
	//unbondings	[]*Unbonding
}

func (a *accountStateImpl) Version() int {
	return a.version
}

func (a *accountStateImpl) SetStake(as state.AccountState, v *common.HexInt) error {

	if a.staked != nil && a.staked.Cmp(&v.Int) == 1 {
		//unstakeAmount :=
		//if len(a.unstakes) < maxUnstake {
		//	append(a.unstakes, newUnstake())
		//} else {
		//
		//}
	}
	a.staked = v

	return nil
}

func (a *accountStateImpl) GetStake() *common.HexInt {
	if a.staked == nil {
		return common.NewHexInt(0)
	}
	return a.staked
}

func (a *accountStateImpl) Bytes() []byte {
	if bs, err := codec.BC.MarshalToBytes(a); err != nil {
		panic(err)
	} else {
		return bs
	}
}

func (a *accountStateImpl) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, a)
	return err
}


func (a *accountStateImpl) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	if err := e2.EncodeMulti(
		a.version,
		a.staked,
	); err != nil {
		return err
	}
	return nil
}

func (a *accountStateImpl) RLPDecodeSelf(d codec.Decoder) error {
	d2, err := d.DecodeList()
	if err != nil {
		return err
	}

	if _, err := d2.DecodeMulti(
		&a.version,
		&a.staked,
	); err != nil {
		return errors.Wrap(err, "Fail to decode accountSnapshot")
	}
	return nil
}

func NewAccountState() AccountState {
	return &accountStateImpl{}
}

type unstake struct {
	amount *common.HexInt
	expireHeight int64
}

func unstakeLockupPeriod() int64 {
	return 0
}

func newUnstake(amount *common.HexInt, blockHeight int64) unstake {
	return unstake{
		amount: amount,
		expireHeight: blockHeight + unstakeLockupPeriod(),
	}
}