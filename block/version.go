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

	"github.com/icon-project/goloop/common/codec"
)

func ReadVersion(r io.Reader) (int, error) {
	d := codec.BC.NewDecoder(r)
	d2, err := d.DecodeList()
	if err != nil {
		return -1, err
	}
	var version int
	err = d2.Decode(&version)
	if err != nil {
		return -1, err
	}
	_ = d.Close()
	return version, nil
}

func PeekVersion(r io.Reader) (int, io.Reader, error) {
	type Peeker interface {
		Peek(n int) ([]byte, error)
	}

	var version int
	if br, ok := r.(io.ReadSeeker); ok {
		pos, err := br.Seek(0, io.SeekCurrent)
		if err != nil {
			return -1, nil, err
		}
		version, err = ReadVersion(br)
		if err != nil {
			return -1, nil, err
		}
		_, err = br.Seek(pos, io.SeekStart)
		if err != nil {
			return -1, nil, err
		}
		return version, br, nil
	}
	if _, ok := r.(Peeker); !ok {
		r = bufio.NewReader(r)
	}
	header, _ := r.(Peeker).Peek(32)
	tr := bytes.NewReader(header)
	version, err := ReadVersion(tr)
	if err != nil {
		return -1, nil, err
	}
	return version, r, nil
}
