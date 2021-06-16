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

package common

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
)

type HexHash []byte

var (
	NullHashBytes  = make([]byte, crypto.HashLen)
	NullHashJSON   = []byte("\"0x" + hex.EncodeToString(NullHashBytes) + "\"")
	NullHashString = "0x"+hex.EncodeToString(NullHashBytes)
)

func (hs HexHash) MarshalJSON() ([]byte, error) {
	if hs == nil {
		return NullHashJSON, nil
	}
	return []byte("\"0x" + hex.EncodeToString(hs) + "\""), nil
}

func (hs *HexHash) UnmarshalJSON(b []byte) error {
	var os *string
	if err := json.Unmarshal(b, &os); err != nil {
		return err
	}
	if os == nil {
		*hs = nil
		return nil
	}
	s := *os
	if len(s) >= 2 && s[0:2] == "0x" {
		s = s[2:]
	}
	if len(s) != crypto.HashLen*2 {
		return errors.IllegalArgumentError.Errorf(
			"IncompatibleHashLength(value=%s)", s)
	}
	if bin, err := hex.DecodeString(s); err != nil {
		return err
	} else {
		if bytes.Equal(bin, NullHashBytes) {
			*hs = nil
		} else {
			*hs = bin
		}
		return nil
	}
}

func (hs HexHash) Bytes() []byte {
	if hs == nil {
		return nil
	}
	return hs[:]
}

func (hs HexHash) String() string {
	if hs == nil {
		return NullHashString
	}
	return "0x" + hex.EncodeToString(hs)
}
