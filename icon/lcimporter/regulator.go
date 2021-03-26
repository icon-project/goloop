/*
 * Copyright 2021 ICON Foundation
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

package lcimporter

import (
	"time"

	"github.com/icon-project/goloop/module"
)

type regulatorImpl struct {
}

func (r *regulatorImpl) MaxTxCount() int {
	return 1000
}

func (r *regulatorImpl) OnPropose(now time.Time) {
	// do nothing
}

func (r *regulatorImpl) CommitTimeout() time.Duration {
	panic("not implemented")
}

func (r *regulatorImpl) MinCommitTimeout() time.Duration {
	panic("not implemented")
}

func (r *regulatorImpl) OnTxExecution(count int, ed time.Duration, fd time.Duration) {
	// do nothing
}

func (r *regulatorImpl) SetBlockInterval(i time.Duration, d time.Duration) {
	// do nothing
}

func NewRegulator() module.Regulator {
	return &regulatorImpl{}
}
