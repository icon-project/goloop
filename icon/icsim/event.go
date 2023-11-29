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

package icsim

import (
	"github.com/icon-project/goloop/module"
)

type Event struct {
	from    module.Address
	indexed [][]byte
	data [][]byte
}

func (e *Event) From() module.Address {
	return e.from
}

func (e *Event) Signature() string {
	return string(e.indexed[0])
}

func (e *Event) Indexed() [][]byte {
	return e.indexed
}

func (e *Event) Data() [][]byte {
	return e.data
}

func NewEvent(from module.Address, indexed, data [][]byte) *Event {
	return &Event{from, indexed, data}
}
