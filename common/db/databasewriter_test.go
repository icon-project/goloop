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

package db

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testFlusher struct {
	err        error
	dbase      Database
	operations []testOperation
}

func (f *testFlusher) Flush() error {
	if f.err != nil {
		return f.err
	}
	for _, op := range f.operations {
		bk, err := f.dbase.GetBucket(op.bk)
		if err != nil {
			return err
		}
		if op.value != nil {
			if err := bk.Set(op.key, op.value); err != nil {
				return err
			}
		} else {
			if err := bk.Delete(op.key); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestWriter_AddPrepareAndFlush(t *testing.T) {
	dbase := newTestDatabase()
	scenario1 := []testOperation {
		{ BytesByHash, []byte("key1"), []byte("value1") },
		{ BytesByHash, []byte("key3"), []byte("value3") },
		{ BytesByHash, []byte("key2"), []byte("value2") },
		{ BytesByHash, []byte("key3"), nil },
	}
	exp1 := []testOperation {
		{ BytesByHash, []byte("key1"), []byte("value1") },
		{ BytesByHash, []byte("key2"), []byte("value2") },
		{ BytesByHash, []byte("key3"), nil },
	}
	w := NewWriter(dbase)
	item1 := &testFlusher{
		dbase: w.Database(),
		operations: scenario1,
	}
	w.Add(item1);
	w.Prepare()
	err := w.Flush()
	assert.NoError(t, err)
	assert.Equal(t, exp1, dbase.record)
}

func TestWriter_AddAndFlushWithError(t *testing.T) {
	dbase := newTestDatabase()
	scenario1 := []testOperation {
		{ BytesByHash, []byte("key1"), []byte("value1") },
		{ BytesByHash, []byte("key3"), []byte("value3") },
		{ BytesByHash, []byte("key2"), []byte("value2") },
		{ BytesByHash, []byte("key3"), nil },
	}
	w := NewWriter(dbase)
	item1 := &testFlusher{
		dbase: w.Database(),
		operations: scenario1,
	}
	w.Add(item1);
	item2 := &testFlusher{
		dbase: w.Database(),
		err: errors.New("failure"),
	}
	w.Add(item2);
	w.Prepare()
	err := w.Flush()
	assert.Error(t, err)
	assert.Equal(t, 0, len(dbase.record))
}

func TestWriter_AddMultipleAndFlush(t *testing.T) {
	dbase := newTestDatabase()
	scenario1 := []testOperation {
		{ BytesByHash, []byte("key1"), []byte("value1") },
		{ BytesByHash, []byte("key3"), []byte("value3") },
		{ BytesByHash, []byte("key2"), []byte("value2") },
		{ BytesByHash, []byte("key3"), nil },
	}
	exp1 := []testOperation {
		{ BytesByHash, []byte("key1"), []byte("value1") },
		{ BytesByHash, []byte("key2"), []byte("value2") },
		{ BytesByHash, []byte("key3"), nil },
	}
	scenario2 := []testOperation {
		{ ChainProperty, []byte("key4"), []byte("value4") },
		{ BytesByHash, []byte("key5"), []byte("value5") },
		{ ChainProperty, []byte("key4"), nil },
	}
	exp2 := []testOperation {
		{ BytesByHash, []byte("key5"), []byte("value5") },
		{ ChainProperty, []byte("key4"), nil },
	}
	w := NewWriter(dbase)
	item1 := &testFlusher{
		dbase: w.Database(),
		operations: scenario1,
	}
	w.Add(item1)
	item2 := &testFlusher{
		dbase: w.Database(),
		operations: scenario2,
	}
	w.Add(item2)
	err := w.Flush()
	assert.NoError(t, err)
	s1, s2 := 0, 0
	for _, rec := range dbase.record {
		if s1 < len(exp1) && reflect.DeepEqual(exp1[s1], rec) {
			s1 += 1
		} else if s2 < len(exp2) && reflect.DeepEqual(exp2[s2], rec) {
			s2 += 1
		} else {
			t.Failed()
		}
	}
	assert.Equal(t, len(exp1), s1)
	assert.Equal(t, len(exp2), s2)
}
