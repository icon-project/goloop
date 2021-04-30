/*
 * Copyright 2020 ICON Foundation
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

package icstate

import (
	"fmt"
	"math/big"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/module"
)

type Unstake struct {
	value *big.Int
	// Unstake is done on OnExecutionEnd of ExpireHeight
	expire int64
}

func NewUnstake(v *big.Int, e int64) *Unstake {
	return &Unstake{
		value:  v,
		expire: e,
	}
}

func (u *Unstake) RLPDecodeSelf(decoder codec.Decoder) error {
	_, err := decoder.DecodeMulti(
		&u.value,
		&u.expire,
	)
	return err
}

func (u *Unstake) RLPEncodeSelf(encoder codec.Encoder) error {
	return encoder.EncodeMulti(
		u.value,
		u.expire,
	)
}

func (u *Unstake) Value() *big.Int {
	return u.value
}

func (u *Unstake) Expire() int64 {
	return u.expire
}

func (u *Unstake) Clone() *Unstake {
	return &Unstake{u.value, u.expire}
}

func (u *Unstake) Equal(u2 *Unstake) bool {
	if u == u2 {
		return true
	}
	return u.value.Cmp(u2.value) == 0 &&
		u.expire == u2.expire
}

func (u Unstake) ToJSON(_ module.JSONVersion, blockHeight int64) interface{} {
	jso := make(map[string]interface{})

	jso["unstake"] = u.value
	jso["unstakeBlockHeight"] = u.expire
	jso["remainingBlocks"] = u.expire - blockHeight

	return jso
}

func (u Unstake) String() string {
	return fmt.Sprintf("Unstake{%d %d}", u.value, u.expire)
}

func (u Unstake) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "Unstake{amount=%d expire=%d}", u.value, u.expire)
			return
		}
		fallthrough
	case 's':
		fmt.Fprint(f, u.String())
	}
}

type Unstakes []*Unstake

func (us Unstakes) Clone() Unstakes {
	if us == nil {
		return nil
	}
	n := make([]*Unstake, len(us))
	for i, u := range us {
		n[i] = u.Clone()
	}
	return n
}

func (us Unstakes) Equal(us2 Unstakes) bool {
	if len(us) != len(us2) {
		return false
	}
	for i, u := range us {
		if !u.Equal(us2[i]) {
			return false
		}
	}
	return true
}

func (us Unstakes) IsEmpty() bool {
	return len(us) == 0
}

// GetUnstakeAmount return unstake Value
func (us Unstakes) GetUnstakeAmount() *big.Int {
	total := new(big.Int)
	for _, u := range us {
		total.Add(total, u.Value())
	}
	return total
}

func (us Unstakes) ToJSON(v module.JSONVersion, blockHeight int64) []interface{} {
	if us.IsEmpty() {
		return nil
	}
	unstakes := make([]interface{}, len(us))

	for idx, p := range us {
		unstakes[idx] = p.ToJSON(v, blockHeight)
	}
	return unstakes
}

func (us *Unstakes) increaseUnstake(v *big.Int, eh int64, sm, revision int) ([]TimerJobInfo, error) {
	if v.Sign() == -1 {
		return nil, errors.Errorf("Invalid unstake Value %v", v)
	}
	tl := make([]TimerJobInfo, 0)
	if len(*us) >= sm {
		// update last entry
		modExpireHeight := false
		lastIndex := len(*us) - 1
		last := (*us)[lastIndex]
		newValue := new(big.Int).Add(last.Value(), v)
		newHeight := eh
		if revision < icmodule.RevisionMultipleUnstakes || eh > last.Expire() {
			modExpireHeight = true
		}
		if modExpireHeight {
			tl = append(tl, TimerJobInfo{JobTypeRemove, last.Expire()})
			tl = append(tl, TimerJobInfo{JobTypeAdd, eh})
			newHeight = eh
		}
		(*us)[lastIndex] = NewUnstake(newValue, newHeight)
	} else {
		unstake := NewUnstake(v, eh)
		unstakes := *us
		index := us.findIndex(eh)
		unstakes = append(unstakes, unstake)
		copy(unstakes[index+1:], unstakes[index:])
		unstakes[index] = unstake
		*us = unstakes
		tl = append(tl, TimerJobInfo{JobTypeAdd, eh})
	}
	return tl, nil
}

func (us Unstakes) findIndex(h int64) int64 {
	for i := len(us) - 1; i >= 0; i-- {
		if h >= us[i].Expire() {
			return int64(i + 1)
		}
	}
	return 0
}

func (us *Unstakes) decreaseUnstake(v *big.Int, expireHeight int64, revision int) ([]TimerJobInfo, error) {
	if v.Sign() == -1 {
		return nil, errors.Errorf("Invalid unstake Value %v", v)
	}
	var tl []TimerJobInfo
	remain := new(big.Int).Set(v) // stakeInc
	uLen := len(*us)
	for i := uLen - 1; i >= 0; i-- {
		u := (*us)[i]
		cmp := remain.Cmp(u.Value())
		switch cmp {
		case 0, 1:
			// Remove an unstake slot
			*us = (*us)[:i]
			tl = append(tl, TimerJobInfo{Type: JobTypeRemove, Height: u.Expire()})
			if cmp == 0 {
				return tl, nil
			} else {
				remain.Sub(remain, u.Value())
			}
		case -1:
			newValue := new(big.Int).Sub(u.Value(), remain)
			newExpire := u.Expire()
			if revision < icmodule.RevisionMultipleUnstakes {
				// must update expire height
				tl = append(tl, TimerJobInfo{Type: JobTypeRemove, Height: u.Expire()})
				tl = append(tl, TimerJobInfo{Type: JobTypeAdd, Height: expireHeight})
				newExpire = expireHeight
			}
			(*us)[i] = NewUnstake(newValue, newExpire)
			return tl, nil
		}
	}
	return tl, nil
}
