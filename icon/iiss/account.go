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
	"github.com/icon-project/goloop/common/log"
)

const (
	accountVersion1 = iota + 1
	accountVersion = accountVersion1
)

type AccountState interface {
	Version() int
	SetStake(v *common.HexInt) error
	GetStake() *common.HexInt
	Bytes() []byte
}

type AccountStateImpl struct {
	version		int
	staked		*common.HexInt
	//delegated	*common.HexInt
	//unstakes	[]*Unstake
	//delegations []*Delegation
	//bonds		[]*Bond
	//unbondings	[]*Unbonding
}

func (a *AccountStateImpl) Version() int {
	return a.version
}

func (a *AccountStateImpl) SetStake(v *common.HexInt) error {
	if a.staked != nil && a.staked.Cmp(&v.Int) == 1 {
		// TODO update unstakes
		log.Debugf("update unbondings")
	}
	a.staked = v

	return nil
}

func (a *AccountStateImpl) GetStake() *common.HexInt {
	if a.staked == nil {
		return common.NewHexInt(0)
	}
	return a.staked
}

func (a *AccountStateImpl) Bytes() []byte {
	if bs, err := codec.BC.MarshalToBytes(a); err != nil {
		panic(err)
	} else {
		return bs
	}
}

func (a *AccountStateImpl) SetBytes(bs []byte) error {
	_, err := codec.BC.UnmarshalFromBytes(bs, a)
	return err
}


func (a *AccountStateImpl) RLPEncodeSelf(e codec.Encoder) error {
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

func (a *AccountStateImpl) RLPDecodeSelf(d codec.Decoder) error {
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