/*
 * Copyright 2022 ICON Foundation
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

package ntm

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIconModule_newAddressFromPubKey(t *testing.T) {
	assert := assert.New(t)
	pk, err := hex.DecodeString("04f309c682bf0d5cc2099e80ad71ed872731138fc9c4df2b38997a9fd4811ed23b7e64777ba926cd51e8ee621ce8f7940c2a091d3480fd7207308c840148e93556")
	assert.NoError(err)
	expAddr, err := hex.DecodeString("005e1e719a335af4f31e6f3f7bd29b6fda1db56a4d")
	assert.NoError(err)
	addr, err := NewIconAddressFromPubKey(pk)
	assert.NoError(err)
	assert.EqualValues(expAddr, addr)
}
