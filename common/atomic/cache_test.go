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

package atomic

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/errors"
)

type Adder struct {
	a   int
	b   int
	sum Cache[int]
}

func (s *Adder) Sum() int {
	return s.sum.Get(func() int {
		var zero atomic.Value
		if s.sum.val != zero {
			panic("creator was called when cache value is not zero")
		}
		return s.a + s.b
	})
}

func (s *Adder) TrySumOK() (int, error) {
	return s.sum.TryGet(func() (int, error) {
		var zero atomic.Value
		if s.sum.val != zero {
			panic("creator was called when cache value is not zero")
		}
		return s.a + s.b, nil
	})
}

func (s *Adder) TrySumFail() (int, error) {
	return s.sum.TryGet(func() (int, error) {
		var zero atomic.Value
		if s.sum.val != zero {
			panic("creator was called when cache value is not zero")
		}
		return 0, errors.New("error")
	})
}

func TestCache_Basics(t *testing.T) {
	adder := Adder{
		a: 1,
		b: 1,
	}
	assert := assert.New(t)
	assert.Equal(2, adder.Sum())
	assert.Equal(2, adder.Sum())

	adder.sum.UnsafePurge()
	var zero atomic.Value
	assert.Equal(zero, adder.sum.val)
	assert.Equal(2, adder.Sum())
	assert.Equal(2, adder.Sum())

	adder.sum.UnsafePurge()
	adder.sum.Set(2)
	assert.Equal(2, adder.Sum())

	adder = Adder{
		a:   1,
		b:   1,
		sum: MakeCache(2),
	}
	assert.Equal(2, adder.Sum())
	assert.Equal(2, adder.Sum())
}

func TestCache_TryGetOK(t *testing.T) {
	assert := assert.New(t)
	adder := Adder{
		a: 1,
		b: 1,
	}
	sum, err := adder.TrySumOK()
	assert.NoError(err)
	assert.Equal(2, sum)
}

func TestCache_TryGetFail(t *testing.T) {
	assert := assert.New(t)
	adder := Adder{
		a: 1,
		b: 1,
	}
	_, err := adder.TrySumFail()
	assert.Error(err)
	sum, err := adder.TrySumOK()
	assert.NoError(err)
	assert.Equal(2, sum)
	sum, err = adder.TrySumOK()
	assert.NoError(err)
	assert.Equal(2, sum)
}
