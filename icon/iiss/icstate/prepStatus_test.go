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
	"github.com/bmizerany/assert"
	"testing"

	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/icon/iiss/icobject"
)

func TestPRepStatusSnapshot_Bytes(t *testing.T) {
	database := icobject.AttachObjectFactory(db.NewMapDB(), newObjectImpl)
	ss1 := newPRepStatusSnapshot(icobject.MakeTag(TypePRepStatus, prepStatusVersion))
	g := PrepGradeCandidate
	ss1.grade = g

	o1 := icobject.New(TypePRepStatus, ss1)
	serialized := o1.Bytes()

	o2 := new(icobject.Object)
	if err := o2.Reset(database, serialized); err != nil {
		t.Errorf("Failed to get object from bytes")
		return
	}

	assert.Equal(t, serialized, o2.Bytes())

	ss2 := ToPRepStatusSnapshot(o2)
	assert.Equal(t, true, ss1.Equal(ss2))
}
