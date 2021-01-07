/*
 * Copyright 2020 ICON Foundation
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package iiss

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
	"github.com/icon-project/goloop/icon/iiss/icstate"
	"github.com/icon-project/goloop/module"
	"github.com/stretchr/testify/assert"
	"testing"
)

// test for handlePrepStatus
func TestUpdatePrepStatus(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), icstate.NewObjectImpl)
	s := icstate.NewStateFromSnapshot(icstate.NewSnapshot(database, nil), false)

	addr1 := common.NewAddressFromString("hx1")
	addr2 := common.NewAddressFromString("hx2")
	status1 := icstate.NewPRepStatus(addr1)
	status2 := icstate.NewPRepStatus(addr2)

	s.AddPRepStatus(status1)
	s.AddPRepStatus(status2)

	addrArray := []module.Address{addr1, addr2}
	voted := []bool{true, false}

	err := handlePrepStatus(s, addrArray, voted)
	assert.NoError(t, err)

	assert.Equal(t, 1, s.GetPRepStatus(addr1).VTotal())
	assert.Equal(t, 1, s.GetPRepStatus(addr2).VTotal())
	assert.Equal(t, 0, s.GetPRepStatus(addr1).VFail())
	assert.Equal(t, 1, s.GetPRepStatus(addr2).VFail())
	assert.Equal(t, 0, s.GetPRepStatus(addr1).VFailCont())
	assert.Equal(t, 1, s.GetPRepStatus(addr2).VFailCont())

	voted2 := []bool{false, false}
	err = handlePrepStatus(s, addrArray, voted2)
	assert.NoError(t, err)

	assert.Equal(t, 2, s.GetPRepStatus(addr1).VTotal())
	assert.Equal(t, 2, s.GetPRepStatus(addr2).VTotal())
	assert.Equal(t, 1, s.GetPRepStatus(addr1).VFail())
	assert.Equal(t, 2, s.GetPRepStatus(addr2).VFail())
	assert.Equal(t, 1, s.GetPRepStatus(addr1).VFailCont())
	assert.Equal(t, 2, s.GetPRepStatus(addr2).VFailCont())

	voted3 := []bool{false, true}
	err = handlePrepStatus(s, addrArray, voted3)
	assert.NoError(t, err)

	assert.Equal(t, 3, s.GetPRepStatus(addr1).VTotal())
	assert.Equal(t, 3, s.GetPRepStatus(addr2).VTotal())
	assert.Equal(t, 2, s.GetPRepStatus(addr1).VFail())
	assert.Equal(t, 2, s.GetPRepStatus(addr2).VFail())
	assert.Equal(t, 2, s.GetPRepStatus(addr1).VFailCont())
	assert.Equal(t, 0, s.GetPRepStatus(addr2).VFailCont())

	voted4 := []bool{false, false}
	err = handlePrepStatus(s, addrArray, voted4)
	assert.NoError(t, err)

	assert.Equal(t, 4, s.GetPRepStatus(addr1).VTotal())
	assert.Equal(t, 4, s.GetPRepStatus(addr2).VTotal())
	assert.Equal(t, 3, s.GetPRepStatus(addr1).VFail())
	assert.Equal(t, 3, s.GetPRepStatus(addr2).VFail())
	assert.Equal(t, 3, s.GetPRepStatus(addr1).VFailCont())
	assert.Equal(t, 1, s.GetPRepStatus(addr2).VFailCont())
}