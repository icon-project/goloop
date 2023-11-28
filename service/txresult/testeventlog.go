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

package txresult

import (
	"bytes"
	"math/big"
	"reflect"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
)

type TestEventLog struct {
	Address module.Address
	Indexed [][]byte
	Data    [][]byte
}

func (ev *TestEventLog) DecodeParams() (signature string, indexed []any, data []any, ret error) {
	if len(ev.Indexed) < 1 {
		return "", nil, nil, errors.IllegalArgumentError.Errorf("NoSignatureToParse")
	}
	signature = string(ev.Indexed[0])
	_, params := DecomposeEventSignature(signature)
	if len(ev.Indexed)+len(ev.Data) != len(params)+1 {
		return "", nil, nil, errors.IllegalArgumentError.New("DataNumberMismatch")
	}
	var decoded *[]any
	for idx, param := range params {
		var input []byte
		if idx < len(ev.Indexed)-1 {
			input = ev.Indexed[idx+1]
			decoded = &indexed
		} else {
			input = ev.Data[idx+1-len(ev.Indexed)]
			decoded = &data
		}
		if input == nil {
			*decoded = append(*decoded, nil)
			continue
		}
		dt := scoreapi.DataTypeOf(param)
		if dt == scoreapi.Unknown {
			return "", nil, nil, errors.IllegalArgumentError.Errorf("UnknownType(dt=%s)", param)
		}
		if value, err := dt.ConvertBytesToAny(input); err != nil {
			return "", nil, nil, err
		} else {
			*decoded = append(*decoded, value)
		}
	}
	return
}

func EqualEventValue(e, r any) bool {
	switch ev := e.(type) {
	case nil:
		return r == nil
	case bool:
		rv, ok := r.(bool)
		if !ok {
			return false
		}
		return rv == ev
	case int64, int, int32, int16, int8:
		rv, ok := r.(*big.Int)
		if !ok {
			return false
		}
		return rv.IsInt64() && rv.Int64() == reflect.ValueOf(ev).Int()
	case uint64, uint, uint32, uint16, uint8:
		rv, ok := r.(*big.Int)
		if !ok {
			return false
		}
		return rv.IsUint64() && rv.Uint64() == reflect.ValueOf(ev).Uint()
	case *big.Int:
		rv, ok := r.(*big.Int)
		if !ok {
			return false
		}
		return rv.Cmp(ev) == 0
	case string:
		rv, ok := r.(string)
		if !ok {
			return false
		}
		return rv == ev
	case []byte:
		rv, ok := r.([]byte)
		if !ok {
			return false
		}
		return bytes.Equal(ev, rv)
	case module.Address:
		rv, ok := r.(module.Address)
		if !ok {
			return false
		}
		return rv.Equal(ev)
	default:
		panic("UnknownTypeForExpectation")
	}
	return false
}

func (ev *TestEventLog) Assert(addr module.Address,
	signature string, indexed, data []any) error {
	s, i, d, err := ev.DecodeParams()
	if err != nil {
		return err
	}
	if !addr.Equal(ev.Address) {
		return errors.InvalidStateError.Errorf(
			"AssertFail: ScoreAddress exp=%v ret=%v",
			addr, ev.Address)
	}
	if signature != s {
		return errors.InvalidStateError.Errorf(
			"AssertFail: Signature exp=%v ret=%v",
			signature, s)
	}
	if len(indexed) != len(i) {
		return errors.InvalidStateError.Errorf(
			"AssertFail: Indexed exp=%v ret=%v",
			indexed, i)
	}
	for idx, e := range indexed {
		if !EqualEventValue(e, i[idx]) {
			return errors.InvalidStateError.Errorf(
				"AssertFail: Indexed[%d] exp=%+v ret=%+v",
				idx, e, i[idx],
			)
		}
	}
	if len(data) != len(d) {
		return errors.InvalidStateError.Errorf(
			"AssertFail: Data exp=%v ret=%v",
			indexed, i)
	}
	for idx, e := range data {
		if !EqualEventValue(e, d[idx]) {
			return errors.InvalidStateError.Errorf(
				"AssertFail: Data[%d] exp=%+v ret=%+v",
				idx, e, d[idx],
			)
		}
	}
	return nil
}
