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
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/log"
)

type transitionResult struct {
	StateHash         []byte
	PatchReceiptHash  []byte
	NormalReceiptHash []byte
	ExtensionData     []byte
}

func newTransitionResultFromBytes(bs []byte) (*transitionResult, error) {
	tresult := new(transitionResult)
	if _, err := codec.UnmarshalFromBytes(bs, tresult); err != nil {
		return nil, err
	}
	return tresult, nil
}

func (tr *transitionResult) RLPEncodeSelf(e codec.Encoder) error {
	e2, err := e.EncodeList()
	if err != nil {
		return err
	}
	err = e2.EncodeMulti(tr.StateHash, tr.PatchReceiptHash, tr.NormalReceiptHash)
	if tr.ExtensionData == nil {
		return err
	}
	return e2.Encode(tr.ExtensionData)
}

func (tr *transitionResult) Bytes() []byte {
	if bs, err := codec.MarshalToBytes(tr); err != nil {
		log.Debug("Fail to marshal transitionResult")
		return nil
	} else {
		return bs
	}
}
