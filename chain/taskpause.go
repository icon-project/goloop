/*
 * Copyright 2023 Parameta Corp
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
	"encoding/json"

	"github.com/icon-project/goloop/common/errors"
)

type taskPause struct {
	chain  *singleChain
	result resultStore
}

var pauseStates = map[State]string{
	Starting: "pausing",
	Started:  "paused",
	Stopping: "stopping paused",
	Failed:   "fail to pause",
}

func (t *taskPause) String() string {
	return "Pause"
}

func (t *taskPause) DetailOf(s State) string {
	if name, ok := pauseStates[s]; ok {
		return name
	} else {
		return s.String()
	}
}

func (t *taskPause) Start() error {
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

func (t *taskPause) _start(c *singleChain) error {
	c.sm.Start()
	//if err := c.cs.Start(); err != nil {
	//	return err
	//}
	c.srv.SetChain(c.cfg.Channel, c)
	if err := c.nm.Start(); err != nil {
		return err
	}
	return nil
}

func (t *taskPause) Stop() {
	t.chain.srv.RemoveChain(t.chain.cfg.Channel)
	t.chain.releaseManagers()
	t.result.SetValue(errors.ErrInterrupted)
}

func (t *taskPause) Wait() error {
	return t.result.Wait()
}

func newTaskPause(chain *singleChain, params json.RawMessage) (chainTask, error) {
	return &taskPause{
		chain: chain,
	}, nil
}

type taskResume struct {
	chain  *singleChain
}

func (t *taskResume) String() string {
	panic("invalid usage")
}

func (t *taskResume) DetailOf(s State) string {
	panic("invalid usage")
}

func (t *taskResume) Start() error {
	panic("invalid usage")
}

func (t *taskResume) Stop() {
	panic("invalid usage")
}

func (t *taskResume) Wait() error {
	panic("invalid usage")
}

func (t *taskResume) Run() error {
	if _, ok := t.chain.task.(*taskPause) ; ok {
		return t.chain.cs.Start()
	} else {
		return errors.InvalidStateError.New("Not in PAUSED state")
	}
}

func newTaskResume(c *singleChain, params json.RawMessage) (chainTask, error) {
	return &taskResume{
		chain: c,
	}, nil
}

func init() {
	registerTaskFactory("pause", newTaskPause)
	registerTaskFactory("resume", newTaskResume)
}
