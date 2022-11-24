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

import "sync"

type resultStore struct {
	have   bool
	result error
	lock   sync.Mutex
	waiter *sync.Cond
}

func (w *resultStore) SetValue(r error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.have {
		return
	}
	w.result = r
	w.have = true
	if w.waiter != nil {
		w.waiter.Broadcast()
	}
}

func (w *resultStore) Wait() error {
	w.lock.Lock()
	defer w.lock.Unlock()

	if w.have {
		return w.result
	}
	if w.waiter == nil {
		w.waiter = sync.NewCond(&w.lock)
	}
	w.waiter.Wait()
	return w.result
}

func (w *resultStore) GetValue() (error, bool) {
	w.lock.Lock()
	defer w.lock.Unlock()

	return w.result, w.have
}
