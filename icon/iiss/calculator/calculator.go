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

package calculator

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
)

type calculator struct {
	log log.Logger

	startHeight int64
	database    db.Database
	back        *icstage.Snapshot
	base        *icreward.Snapshot
	global      icstage.Global
	temp        *icreward.State
	stats       *Stats

	lock    sync.Mutex
	waiters []*sync.Cond
	err     error
	result  *icreward.Snapshot
}

func (c *calculator) Result() *icreward.Snapshot {
	return c.result
}

func (c *calculator) StartHeight() int64 {
	return c.startHeight
}

func (c *calculator) TotalReward() *big.Int {
	return c.stats.Total()
}

func (c *calculator) Back() *icstage.Snapshot {
	return c.back
}

func (c *calculator) Base() *icreward.Snapshot {
	return c.base
}

func (c *calculator) Temp() *icreward.State {
	return c.temp
}

func (c *calculator) Global() icstage.Global {
	return c.global
}

func (c *calculator) WaitResult(blockHeight int64) error {
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

func (c *calculator) setResult(result *icreward.Snapshot, err error) {
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

func (c *calculator) Stats() *Stats {
	return c.stats
}

func (c *calculator) Logger() log.Logger {
	return c.log
}

func (c *calculator) Stop() {
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

func (c *calculator) IsRunningFor(dbase db.Database, back, reward []byte) bool {
	return c.database == dbase &&
		bytes.Equal(c.back.Bytes(), back) &&
		bytes.Equal(c.base.Bytes(), reward)
}

func (c *calculator) run() error {
	var err error
	defer func() {
		if err != nil {
			c.setResult(nil, err)
		}
	}()

	if err = c.prepare(); err != nil {
		return icmodule.CalculationFailedError.Wrapf(err, "Failed to prepare calculator")
	}

	iv := c.global.GetIISSVersion()
	var r Reward
	switch iv {
	case icstate.IISSVersion2, icstate.IISSVersion3:
		r = NewIISS3Reward(c)
	case icstate.IISSVersion4:
		if r, err = NewIISS4Reward(c); err != nil {
			return icmodule.CalculationFailedError.Wrapf(err, "Failed to init IISS4 reward")
		}
	default:
		return icmodule.CalculationFailedError.Wrapf(err, "invalid IISS version")
	}
	if err = r.Calculate(); err != nil {
		return icmodule.CalculationFailedError.Wrapf(err, "Failed to calculate reward")
	}

	if err = c.postWork(); err != nil {
		return icmodule.CalculationFailedError.Wrapf(err, "Failed to do post work of calculator")
	}

	c.log.Infof("Calculation statistics: %s", c.stats)
	c.setResult(c.temp.GetSnapshot(), nil)
	return nil
}

func (c *calculator) prepare() error {
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

func (c *calculator) processClaim() error {
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

func (c *calculator) postWork() (err error) {
	// write BTP data to temp. Use BTP data in the next term
	if err = c.processBTP(); err != nil {
		return err
	}
	// update Voted.commissionRate of temp. Use updated commission rate in the next term
	if err = c.processCommissionRate(); err != nil {
		return err
	}
	return nil
}

func (c *calculator) processBTP() error {
	for iter := c.back.Filter(icstage.BTPKey.Build()); iter.Has(); iter.Next() {
		o, _, err := iter.Get()
		if err != nil {
			return err
		}
		obj := o.(*icobject.Object)
		switch obj.Tag().Type() {
		case icstage.TypeBTPDSA:
			value := icstage.ToBTPDSA(o)
			dsa, err := c.temp.GetDSA()
			if err != nil {
				return err
			}
			nDSA := dsa.Updated(value.Index())
			if err = c.temp.SetDSA(nDSA); err != nil {
				return err
			}
		case icstage.TypeBTPPublicKey:
			value := icstage.ToBTPPublicKey(o)
			pubKey, err := c.temp.GetPublicKey(value.From())
			if err != nil {
				return nil
			}
			nPubKey := pubKey.Updated(value.Index())
			if err = c.temp.SetPublicKey(value.From(), nPubKey); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *calculator) processCommissionRate() error {
	prefix := icstage.CommissionRateKey.Build()
	for iter := c.back.Filter(prefix); iter.Has(); iter.Next() {
		o, key, err := iter.Get()
		if err != nil {
			return err
		}

		obj := o.(*icobject.Object)
		if obj.Tag().Type() == icstage.TypeCommissionRate {
			keySplit, err := containerdb.SplitKeys(key)
			if err != nil {
				return err
			}
			addr, err := common.NewAddress(keySplit[1])
			if err != nil {
				return nil
			}
			cr := icstage.ToCommissionRate(o)
			voted, err := c.temp.GetVoted(addr)
			if err != nil {
				return nil
			}
			if voted == nil {
				return nil
			}
			nVoted := voted.Clone()
			nVoted.SetCommissionRate(cr.Value())
			err = c.temp.SetVoted(addr, nVoted)
			if err != nil {
				return nil
			}
		}
	}
	return nil
}

const InitBlockHeight = -1

func New(database db.Database, back *icstage.Snapshot, reward *icreward.Snapshot, logger log.Logger) *calculator {
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
	c := &calculator{
		database:    database,
		back:        back,
		base:        reward,
		temp:        icreward.NewStateFromSnapshot(reward),
		log:         logger,
		global:      global,
		startHeight: startHeight,
		stats:       NewStats(),
	}
	if startHeight != InitBlockHeight {
		go c.run()
	}
	return c
}
