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

package txresult

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/module"
)

func TestFeeDetail_Basic(t *testing.T) {
	var fd1 feeDetail
	var ok bool

	assert.False(t, fd1.Has())

	ok = fd1.AddPayment(
		common.MustNewAddressFromString("hx9e62261634efa9733f37ef449fd32209de272453"),
		big.NewInt(1000),
	)

	assert.True(t, ok)
	assert.True(t, fd1.Has())

	jso, err := fd1.ToJSON(module.JSONVersionLast)
	assert.NoError(t, err)

	msg, err := json.Marshal(jso)
	assert.NoError(t, err)
	fmt.Printf("%s\n", msg)

	bs, err := codec.BC.MarshalToBytes(fd1)
	assert.NoError(t, err)

	var fd2 feeDetail
	_, err = codec.BC.UnmarshalFromBytes(bs, &fd2)
	assert.NoError(t, err)

	assert.Equal(t, fd1, fd2)
}
