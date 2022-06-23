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

package block

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
)

func TestPeekVersionBytesReader(t *testing.T) {
	hf := V2HeaderFormat{
		Version: 1,
		Result:  []byte{1, 2},
	}
	bs := codec.MustMarshalToBytes(hf)

	v, r, err := PeekVersion(bytes.NewReader(bs))
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	var hf2 V2HeaderFormat
	err = codec.Unmarshal(r, &hf2)
	assert.Nil(t, err)
	assert.Equal(t, hf, hf2)
}

func TestPeekVersionBufIOReader(t *testing.T) {
	hf := V2HeaderFormat{
		Version: 1,
		Result:  []byte{1, 2},
	}
	bs := codec.MustMarshalToBytes(hf)

	v, r, err := PeekVersion(bufio.NewReader(bytes.NewReader(bs)))
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	var hf2 V2HeaderFormat
	err = codec.Unmarshal(r, &hf2)
	assert.Nil(t, err)
	assert.Equal(t, hf, hf2)
}

type testReader struct {
	bs []byte
}

func (r *testReader) Read(p []byte) (n int, err error) {
	if len(r.bs) == 0 {
		return 0, io.EOF
	}
	n = copy(p, r.bs)
	r.bs = r.bs[n:]
	return n, nil
}

func TestPeekVersionTestReader(t *testing.T) {
	hf := V2HeaderFormat{
		Version: 1,
		Result:  []byte{1, 2},
	}
	bs := codec.MustMarshalToBytes(hf)

	v, r, err := PeekVersion(&testReader{bs})
	assert.Nil(t, err)
	assert.Equal(t, 1, v)
	var hf2 V2HeaderFormat
	err = codec.Unmarshal(r, &hf2)
	assert.Nil(t, err)
	assert.Equal(t, hf, hf2)
}
