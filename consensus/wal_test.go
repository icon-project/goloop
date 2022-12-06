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
	"encoding/binary"
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
		FileLimit:            12*97 + 1,
		TotalLimit:           12 * 5000,
		HousekeepingInterval: time.Millisecond * 50,
	})
	defer func() {
		_ = os.RemoveAll(base)
	}()
	assert.NoError(t, err)
	const iterations = 10000
	for i := 0; i < iterations; i++ {
		var buf [4]byte
		binary.BigEndian.PutUint32(buf[:], uint32(i))
		_, err = ww.WriteBytes(buf[:])
		assert.NoError(t, err)
		//t.Logf("Write %v", i)
		if i%150 == 0 {
			time.Sleep(time.Millisecond * 50)
		}
		if i%100 == 0 {
			err = ww.Sync()
			assert.NoError(t, err)
		}
	}
	err = ww.Close()
	assert.NoError(t, err)

	wr, err := consensus.OpenWALForRead(id)
	assert.NoError(t, err)
	bs, err := wr.ReadBytes()
	assert.NoError(t, err)
	v := binary.BigEndian.Uint32(bs)
	t.Logf("Read from %v", v)
	for i := v + 1; i < iterations; i++ {
		bs, err := wr.ReadBytes()
		assert.NoError(t, err)
		v := binary.BigEndian.Uint32(bs)
		//t.Logf("Read %v", v)
		assert.EqualValues(t, int(i), int(v))
	}
	err = wr.Close()
	assert.NoError(t, err)
}
