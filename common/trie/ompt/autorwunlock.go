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
 *
 */

package ompt

import "sync"

type LockState int

const (
	LockNone LockState = iota
	LockRead
	LockWrite
)

type AutoRWUnlock struct {
	lock    *sync.RWMutex
	state   LockState
}

func (l *AutoRWUnlock) Migrate() {
	switch l.state {
	case LockNone:
		l.lock.Lock()
		l.state = LockWrite
		return
	case LockRead:
		l.lock.RUnlock()
		l.lock.Lock()
		l.state = LockWrite
		return
	case LockWrite:
		return
	}
}

func (l *AutoRWUnlock) Unlock() {
	switch l.state {
	case LockNone:
		return
	case LockRead:
		l.state = LockNone
		l.lock.RUnlock()
	case LockWrite:
		l.state = LockNone
		l.lock.Unlock()
	}
}

func RLock(l *sync.RWMutex) AutoRWUnlock {
	l.RLock()
	return AutoRWUnlock{
		lock: l,
		state: LockRead,
	}
}

