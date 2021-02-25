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

package icstate

import (
	"math/big"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type Voting interface {
	To() module.Address
	Amount() *big.Int
}

type VotingIterator interface {
	Has() bool
	Next() error
	Get() (Voting, error)
}

type votingIterator struct {
	index   int
	votings []Voting
}

func (i *votingIterator) Has() bool {
	return i.index < len(i.votings)
}

func (i *votingIterator) Next() error {
	if i.index < len(i.votings) {
		i.index += 1
		return nil
	} else {
		return errors.ErrInvalidState
	}
}

func (i *votingIterator) Get() (Voting, error) {
	if i.index < len(i.votings) {
		return i.votings[i.index], nil
	} else {
		return nil, errors.ErrInvalidState
	}
}

func NewVotingIterator(votings []Voting) *votingIterator {
	return &votingIterator{
		index: 0,
		votings: votings,
	}
}
