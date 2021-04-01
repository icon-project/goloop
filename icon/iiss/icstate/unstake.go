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
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/intconv"
	"github.com/icon-project/goloop/module"
)

type Unstake struct {
	Amount *big.Int
	// Unstake is done on OnExecutionEnd of ExpireHeight
	ExpireHeight int64
}

func newUnstake(amount *big.Int, expireHeight int64) *Unstake {
	return &Unstake{
		Amount:       new(big.Int).Set(amount),
		ExpireHeight: expireHeight,
	}
}

func (u *Unstake) Clone() *Unstake {
	return newUnstake(u.Amount, u.ExpireHeight)
}

func (u *Unstake) Equal(u2 *Unstake) bool {
	if u == u2 {
		return true
	}
	return u.Amount.Cmp(u2.Amount) == 0 &&
		u.ExpireHeight == u2.ExpireHeight
}

func (u Unstake) ToJSON(_ module.JSONVersion) interface{} {
	jso := make(map[string]interface{})

	jso["unstake"] = intconv.FormatBigInt(u.Amount)
	jso["expireBlockHeight"] = intconv.FormatInt(u.ExpireHeight)

	return jso
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

func (us Unstakes) Has() bool {
	return len(us) > 0
}

// GetUnstakeAmount return unstake Value
func (us Unstakes) GetUnstakeAmount() *big.Int {
	total := new(big.Int)
	for _, u := range us {
		total.Add(total, u.Amount)
	}
	return total
}

func (us Unstakes) ToJSON(v module.JSONVersion) []interface{} {
	if us.Has() == false {
		return nil
	}
	unstakes := make([]interface{}, len(us))

	for idx, p := range us {
		unstakes[idx] = p.ToJSON(v)
	}
	return unstakes
}

func (us *Unstakes) increaseUnstake(v *big.Int, eh int64, sm int) ([]TimerJobInfo, error) {
	if v.Sign() == -1 {
		return nil, errors.Errorf("Invalid unstake Value %v", v)
	}
	tl := make([]TimerJobInfo, 0)
	if len(*us) >= sm {
		// update last entry
		lastIndex := len(*us) - 1
		last := (*us)[lastIndex]
		last.Amount.Add(last.Amount, v)
		tl = append(tl, TimerJobInfo{JobTypeRemove, last.ExpireHeight})
		tl = append(tl, TimerJobInfo{JobTypeAdd, eh})
		if eh > last.ExpireHeight {
			last.ExpireHeight = eh
		}
	} else {
		unstake := newUnstake(v, eh)
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
		if h >= us[i].ExpireHeight {
			return int64(i + 1)
		}
	}
	return 0
}

func (us *Unstakes) decreaseUnstake(v *big.Int) ([]TimerJobInfo, error) {
	if v.Sign() == -1 {
		return nil, errors.Errorf("Invalid unstake Value %v", v)
	}
	var tl []TimerJobInfo
	remain := new(big.Int).Set(v) // stakeInc
	uLen := len(*us)
	for i := uLen - 1; i >= 0; i-- {
		u := (*us)[i]
		cmp := remain.Cmp(u.Amount)
		switch cmp {
		case 0, 1:
			// Remove an unstake slot
			*us = (*us)[:i]
			tl = append(tl, TimerJobInfo{Type: JobTypeRemove, Height: u.ExpireHeight})
			if cmp == 0 {
				return tl, nil
			} else {
				remain.Sub(remain, u.Amount)
			}
		case -1:
			u.Amount.Sub(u.Amount, remain)
			return tl, nil
		}
	}
	return tl, nil
}
