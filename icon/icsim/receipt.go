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

package icsim

const (
	Failure = 0
	Success = 1
)

type Receipt interface {
	BlockHeight() int64
	Status() int
	Error() error
	Events() []*Event
}

type receipt struct {
	blockHeight int64
	status int
	err error
	events []*Event
}

func (r *receipt) BlockHeight() int64 {
	return r.blockHeight
}

func (r *receipt) Status() int {
	return r.status
}

func (r *receipt) Error() error {
	return r.err
}

func (r *receipt) Events() []*Event{
	return r.events
}

func NewReceipt(blockHeight int64, err error, events []*Event) Receipt {
	status := Success
	if err != nil {
		status = Failure
	}
	return &receipt{
		blockHeight: blockHeight,
		status: status,
		err: err,
		events: events,
	}
}
