/*
 * Copyright 2023 ICON Foundation
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

package rewards

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icreward"
	"github.com/icon-project/goloop/icon/iiss/icstage"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
)

type Calculator struct {
	log log.Logger

	startHeight int64
	database    db.Database
	back        *icstage.Snapshot
	base        *icreward.Snapshot
	global      icstage.Global
	temp        *icreward.State
	stats       *statistics

	lock    sync.Mutex
	waiters []*sync.Cond
	err     error
	result  *icreward.Snapshot
}

func (c *Calculator) Result() *icreward.Snapshot {
	return c.result
}

func (c *Calculator) StartHeight() int64 {
	return c.startHeight
}

func (c *Calculator) TotalReward() *big.Int {
	return c.stats.TotalReward()
}

func (c *Calculator) Back() *icstage.Snapshot {
	return c.back
}

func (c *Calculator) Base() *icreward.Snapshot {
	return c.base
}

func (c *Calculator) Temp() *icreward.State {
	return c.temp
}

func (c *Calculator) Error() error {
	return c.err
}

func (c *Calculator) WaitResult(blockHeight int64) error {
	if c.startHeight == InitBlockHeight {
		return nil
	}
	if c.startHeight != blockHeight {
		return errors.InvalidStateError.Errorf("Calculator(height=%d,exp=%d)",
			c.startHeight, blockHeight)
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.err == nil && c.result == nil {
		cond := sync.NewCond(&c.lock)
		c.waiters = append(c.waiters, cond)
		cond.Wait()
	}
	return c.err
}

func (c *Calculator) setResult(result *icreward.Snapshot, err error) {
	if result == nil && err == nil {
		c.log.Panicf("InvalidParameters(result=%+v, err=%+v)")
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	// it's already interrupted.
	if c.err != nil {
		return
	}

	c.result = result
	c.err = err
	for _, cond := range c.waiters {
		cond.Signal()
	}
	c.waiters = nil
}

func (c *Calculator) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.err == nil && c.result == nil {
		c.err = errors.ErrInterrupted
		for _, w := range c.waiters {
			w.Signal()
		}
		c.waiters = nil
	}
}

func (c *Calculator) IsRunningFor(dbase db.Database, back, reward []byte) bool {
	return c.database == dbase &&
		bytes.Equal(c.back.Bytes(), back) &&
		bytes.Equal(c.base.Bytes(), reward)
}

func (c *Calculator) run() (err error) {
	defer func() {
		if err != nil {
			c.setResult(nil, err)
		}
	}()

	if err = c.prepare(); err != nil {
		err = icmodule.CalculationFailedError.Wrapf(err, "Failed to prepare calculator")
		return
	}

	switch c.global.GetIISSVersion() {
	case icstate.IISSVersion2, icstate.IISSVersion3:
		if err = c.calculateRewardV3(); err != nil {
			return
		}
	case icstate.IISSVersion4:
		// TODO IISS4
	}

	if err = c.postWork(); err != nil {
		err = icmodule.CalculationFailedError.Wrapf(err, "Failed to do post work of calculator")
		return
	}

	c.log.Infof("Calculation statistics: Total=%d BlockProduce=%s Voted=%s Voting=%s",
		c.stats.TotalReward(), c.stats.BlockProduce(), c.stats.Voted(), c.stats.Voting())

	c.setResult(c.temp.GetSnapshot(), nil)
	return nil
}

func (c *Calculator) prepare() error {
	var err error
	c.log.Infof("Start calculation %d", c.startHeight)
	c.log.Infof("Global Option: %+v", c.global)

	// write claim data to temp
	if err = c.processClaim(); err != nil {
		return err
	}

	// replay BugDisabledPRep
	if err = c.replayBugDisabledPRep(); err != nil {
		return err
	}
	return nil
}

func (c *Calculator) processClaim() error {
	for iter := c.back.Filter(icstage.IScoreClaimKey.Build()); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}
		obj := o.(*icobject.Object)
		if obj.Tag().Type() == icstage.TypeIScoreClaim {
			claim := icstage.ToIScoreClaim(o)
			keySplit, err := containerdb.SplitKeys(key)
			if err != nil {
				return nil
			}
			addr, err := common.NewAddress(keySplit[1])
			if err != nil {
				return nil
			}
			iScore, err := c.temp.GetIScore(addr)
			if err != nil {
				return nil
			}
			nIScore := iScore.Subtracted(claim.Value())
			if nIScore.Value().Sign() == -1 {
				return errors.Errorf("Invalid negative I-Score for %s. %+v - %+v = %+v", addr, iScore, claim, nIScore)
			}
			c.log.Tracef("Claim %s. %+v - %+v = %+v", addr, iScore, claim, nIScore)
			if err = c.temp.SetIScore(addr, nIScore); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Calculator) updateIScore(addr module.Address, reward *big.Int, t RewardType) error {
	iScore, err := c.temp.GetIScore(addr)
	if err != nil {
		return err
	}
	nIScore := iScore.Added(reward)
	if err = c.temp.SetIScore(addr, nIScore); err != nil {
		return err
	}
	c.log.Tracef("Update IScore %s by %d: %+v + %s = %+v", addr, t, iScore, reward, nIScore)

	switch t {
	case TypeBlockProduce:
		c.stats.IncreaseBlockProduce(reward)
	case TypeVoted:
		c.stats.IncreaseVoted(reward)
	case TypeVoting:
		c.stats.IncreaseVoting(reward)
	}
	return nil
}

const InitBlockHeight = -1

func NewCalculator(database db.Database, back *icstage.Snapshot, reward *icreward.Snapshot, logger log.Logger) *Calculator {
	var err error
	var global icstage.Global
	var startHeight int64

	global, err = back.GetGlobal()
	if err != nil {
		logger.Errorf("Failed to get Global values for calculator. %+v", err)
		return nil
	}
	if global == nil {
		// back has no global at first term
		startHeight = InitBlockHeight
	} else {
		startHeight = global.GetStartHeight()
	}
	c := &Calculator{
		database:    database,
		back:        back,
		base:        reward,
		temp:        icreward.NewStateFromSnapshot(reward),
		log:         logger,
		global:      global,
		startHeight: startHeight,
		stats:       newStatistics(),
	}
	if startHeight != InitBlockHeight {
		go c.run()
	}
	return c
}
