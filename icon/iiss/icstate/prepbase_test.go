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

package icstate

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestPRepBase_Bytes(t *testing.T) {
	owner, err := common.NewAddress(make([]byte, common.AddressBytes, common.AddressBytes))
	if err != nil {
		t.Errorf("Failed to create an address")
	}

	database := icobject.AttachObjectFactory(db.NewMapDB(), NewObjectImpl)
	ss1 := newPRepBaseWithTag(icobject.MakeTag(TypePRepBase, prepVersion))
	n := "name"
	ss1.SetOwner(owner)
	ss1.name = n

	o1 := icobject.New(TypePRepBase, ss1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ss2 := ToPRepBase(o2, owner)
	assert.Equal(t, true, ss1.Equal(ss2))
	assert.Equal(t, true, ss1.owner.Equal(owner))
	assert.Equal(t, true, ss2.owner.Equal(owner))
}
