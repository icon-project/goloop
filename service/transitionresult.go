/*
 * Copyright 2020 ICON Foundation
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

package service

import (
	"io"

	"github.com/icon-project/goloop/chain/base"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/containerdb"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoredb"
	"github.com/icon-project/goloop/service/state"
)

type transitionResult struct {
	StateHash         []byte
	PatchReceiptHash  []byte
	NormalReceiptHash []byte
	ExtensionData     []byte
	BTPData           []byte
}

const (
	ExFlagBTPData = 1 << iota
)

func newTransitionResultFromBytes(bs []byte) (*transitionResult, error) {
	tresult := new(transitionResult)
	if len(bs) > 0 {
		if _, err := codec.UnmarshalFromBytes(bs, tresult); err != nil {
			return nil, err
		}
	}
	return tresult, nil
}

func (tr *transitionResult) getExtensionFlags() int64 {
	var flags int64 = 0
	if len(tr.BTPData) > 0 {
		flags |= ExFlagBTPData
	}
	return flags
}

func (tr *transitionResult) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	err = e2.EncodeMulti(tr.StateHash, tr.PatchReceiptHash, tr.NormalReceiptHash)
	if err != nil {
		return err
	}
	exFlags := tr.getExtensionFlags()
	if tr.ExtensionData == nil && exFlags == 0 {
		return nil
	}
	if err := e2.Encode(tr.ExtensionData); err != nil {
		return err
	}
	if exFlags != 0 {
		if err := e2.Encode(exFlags); err != nil {
			return err
		}
		if (exFlags & ExFlagBTPData) != 0 {
			if err := e2.Encode(tr.BTPData); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tr *transitionResult) RLPDecodeSelf(e codec.Decoder) error {
	d2, err := e.DecodeList()
	if err != nil {
		return err
	}
	var exFlags int64
	if _, err := d2.DecodeMulti(
		&tr.StateHash, &tr.PatchReceiptHash, &tr.NormalReceiptHash,
		&tr.ExtensionData, &exFlags); err == nil {
		if exFlags == 0 {
			return InvalidResultError.Errorf("UnnecessaryExFlag")
		}
		if exFlags&ExFlagBTPData != 0 {
			if err := d2.Decode(&tr.BTPData); err != nil {
				return InvalidResultError.Errorf("NoBTPDigest")
			}
			exFlags ^= ExFlagBTPData
		}
		if exFlags != 0 {
			return InvalidResultError.Errorf("UnresolvedExtensionFlags(flag=%#x", exFlags)
		}
	} else if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (tr *transitionResult) Bytes() []byte {
	if bs, err := codec.MarshalToBytes(tr); err != nil {
		log.Debug("Fail to marshal transitionResult")
		return nil
	} else {
		return bs
	}
}

func NewWorldSnapshot(database db.Database, plt base.Platform, result []byte, vl module.ValidatorList) (state.WorldSnapshot, error) {
	return newWorldSnapshot(database, plt, result, vl)
}

func NewBTPContext(dbase db.Database, result []byte) (state.BTPContext, error) {
	wss, err := NewWorldSnapshot(dbase, nil, result, nil)
	if err != nil {
		return nil, err
	}
	acss := wss.GetAccountSnapshot(state.SystemID)
	var as containerdb.BytesStoreState
	if acss == nil {
		as = containerdb.EmptyBytesStoreState
	} else {
		as = scoredb.NewStateStoreWith(acss)
	}
	return state.NewBTPContext(nil, as), nil
}

func BTPDigestHashFromResult(result []byte) ([]byte, error) {
	r, err := newTransitionResultFromBytes(result)
	if err != nil {
		return nil, err
	}
	return r.BTPData, nil
}
