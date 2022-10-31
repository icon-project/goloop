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

package consensus_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/consensus"
)

func TestWAL(t *testing.T) {
	base, err := os.MkdirTemp("", "goloop-waltest")
	assert.NoError(t, err)
	id := base + "/testwal"
	ww, err := consensus.OpenWALForWrite(id, &consensus.WALConfig{
		FileLimit:            1024 * 400,
		TotalLimit:           1024 * 400 * 10,
		HousekeepingInterval: time.Millisecond * 50,
	})
	defer func() {
		_ = os.RemoveAll(base)
	}()
	assert.NoError(t, err)
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		err = consensus.WALWriteObject(ww, i)
		assert.NoError(t, err)
		//t.Logf("Write %v", i)
		/*
			var k int
			for j := 0; j < 100000; j++ {
				k += j
			}
			t.Logf("k=%v\n", k)
		*/
		//time.Sleep(time.Microsecond * 1)
		if i%100 == 0 {
			err = ww.Sync()
			assert.NoError(t, err)
		}
	}
	err = ww.Close()
	assert.NoError(t, err)

	wr, err := consensus.OpenWALForRead(id)
	assert.NoError(t, err)
	for i := 0; i < iterations; i++ {
		var v int
		_, err := consensus.WALReadObject(wr, &v)
		assert.NoError(t, err)
		//t.Logf("Read %v", v)
		assert.EqualValues(t, i, v)
	}
	err = wr.Close()
	assert.NoError(t, err)
}
