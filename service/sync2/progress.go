/*
 * Copyright 2022 ICON Foundation
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

package sync2

import "sync"

type progressItem struct {
	r, u int
}

type progressSum struct {
	lock  sync.Mutex
	state []progressItem
	cb    ProgressCallback
	r, u  int
}

func newProgressSum(n int, cb ProgressCallback) *progressSum {
	return &progressSum{
		state: make([]progressItem, n),
		cb:    cb,
	}
}

func (p *progressSum) onProgress(i, r, u int) error {
	if p.cb != nil {
		p.lock.Lock()
		defer p.lock.Unlock()

		item := &p.state[i]
		p.r += r - item.r
		p.u += u - item.u
		item.r = r
		item.u = u
		return p.cb(p.r, p.u)
	} else {
		return nil
	}
}

func (p *progressSum) callbackOf(i int) ProgressCallback {
	if p.cb != nil {
		return func(r, u int) error {
			return p.onProgress(i, r, u)
		}
	}
	return nil
}
