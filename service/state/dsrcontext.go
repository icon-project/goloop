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

package state

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
)

type dsValidators struct {
	vl    module.ValidatorList
	bytes []byte
}

func (c *dsValidators) AddressOf(signer []byte) module.Address {
	addr := common.NewAddressWithTypeAndID(false, signer)
	if idx := c.vl.IndexOf(addr); idx < 0 {
		return nil
	} else {
		return addr
	}
}

func (c *dsValidators) Bytes() []byte {
	if c.bytes == nil {
		c.bytes = codec.BC.MustMarshalToBytes([][]byte{c.vl.Bytes()})
	}
	return c.bytes
}

func (c *dsValidators) Hash() []byte {
	return c.vl.Hash()
}

func decodeDoubleSignContext(t string, d []byte) (module.DoubleSignContext, error) {
	switch t {
	case module.DSTProposal, module.DSTVote:
		var proof [][]byte
		remain, err := codec.BC.UnmarshalFromBytes(d, &proof)
		if err != nil {
			return nil, errors.IllegalArgumentError.Wrap(err, "InvalidFormat")
		} else if len(remain) > 0 {
			return nil, errors.IllegalArgumentError.Errorf("InvalidTrailingBytes(n=%d)", len(remain))
		}
		if len(proof) != 1 {
			return nil, errors.IllegalArgumentError.New("InvalidContextData")
		}
		vl, err := NewValidatorListFromBytes(proof[0])
		if err != nil {
			return nil, err
		}
		return &dsValidators{vl, d}, nil
	default:
		return nil, errors.IllegalArgumentError.Errorf("InvalidType(type=%s)", t)
	}
}

type dsContextRoot struct {
	vl module.ValidatorList
}

func (d *dsContextRoot) Hash() []byte {
	return d.vl.Hash()
}

func (d *dsContextRoot) ContextOf(tn string) (module.DoubleSignContext, error) {
	switch tn {
	case module.DSTProposal, module.DSTVote:
		return &dsValidators {
			vl: d.vl,
		}, nil
	default:
		return nil, errors.IllegalArgumentError.Errorf("UnknownType(tn=%s)", tn)
	}
}

func getDoubleSignContextRootOf(ws WorldState, revision module.Revision) (module.DoubleSignContextRoot, error) {
	if revision.Has(module.ReportDoubleSign) {
		vl, err := ToValidatorList(ws.GetValidatorState().GetSnapshot())
		if err != nil {
			return nil, err
		}
		return &dsContextRoot{
			vl: vl,
		}, nil
	}
	return nil, nil
}