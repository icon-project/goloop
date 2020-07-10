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
	"os"
	"path"
)

type taskReset struct {
	chain  *singleChain
	result resultStore
}

var resetStates = map[State]string{
	Starting: "reset starting",
	Started:  "reset started",
	Stopping: "reset stopping",
	Failed:   "reset failed",
	Finished: "reset finished",
}

func (t *taskReset) String() string {
	return "Reset"
}

func (t *taskReset) DetailOf(s State) string {
	if name, ok := resetStates[s]; ok {
		return name
	} else {
		return s.String()
	}
}

func (t *taskReset) Start() error {
	go t.doReset()
	return nil
}

func (t *taskReset) doReset() {
	err := t._reset()
	t.result.SetValue(err)
}

func (t *taskReset) _reset() error {
	c := t.chain
	chainDir := c.cfg.AbsBaseDir()

	c.releaseDatabase()
	defer c.ensureDatabase()

	DBDir := path.Join(chainDir, DefaultDBDir)
	if err := os.RemoveAll(DBDir); err != nil {
		return err
	}
	CacheDir := path.Join(chainDir, DefaultCacheDir)
	if err := os.RemoveAll(CacheDir); err != nil {
		return err
	}

	WALDir := path.Join(chainDir, DefaultWALDir)
	if err := os.RemoveAll(WALDir); err != nil {
		return err
	}

	ContractDir := path.Join(chainDir, DefaultContractDir)
	if err := os.RemoveAll(ContractDir); err != nil {
		return err
	}

	return nil
}

func (t *taskReset) Stop() {
	// do nothing (it's hard to stop )
}

func (t *taskReset) Wait() error {
	return t.result.Wait()
}

func newTaskReset(chain *singleChain) chainTask {
	return &taskReset{
		chain: chain,
	}
}
