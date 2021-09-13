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

package chain

import (
	"github.com/icon-project/goloop/common/errors"
)

type taskConsensus struct {
	chain  *singleChain
	result resultStore
}

var consensusStates = map[State]string{
	Starting: "starting",
	Started:  "started",
	Stopping: "stopping",
	Failed:   "failed",
}

func (t *taskConsensus) String() string {
	return "Consensus"
}

func (t *taskConsensus) DetailOf(s State) string {
	if name, ok := consensusStates[s]; ok {
		return name
	} else {
		return s.String()
	}
}

func (t *taskConsensus) Start() error {
	if err := t.chain.prepareManagers(); err != nil {
		t.result.SetValue(err)
		return err
	}
	if err := t._start(t.chain); err != nil {
		t.chain.releaseManagers()
		t.result.SetValue(err)
		return err
	}
	return nil
}

func (t *taskConsensus) _start(c *singleChain) error {
	c.sm.Start()
	if err := c.cs.Start(); err != nil {
		return err
	}
	c.srv.SetChain(c.cfg.Channel, c)
	if err := c.nm.Start(); err != nil {
		return err
	}
	return nil
}

func (t *taskConsensus) Stop() {
	t.chain.srv.RemoveChain(t.chain.cfg.Channel)
	t.chain.releaseManagers()
	t.result.SetValue(errors.ErrInterrupted)
}

func (t *taskConsensus) Wait() error {
	return t.result.Wait()
}

func newTaskConsensus(chain *singleChain) chainTask {
	return &taskConsensus{
		chain: chain,
	}
}
