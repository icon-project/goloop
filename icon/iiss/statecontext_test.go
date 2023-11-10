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
 */

package iiss

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/icon/icmodule"
)

func TestStateContext_Flags(t *testing.T) {
	es := newDummyExtensionState(t)

	args := []struct{
		flags icmodule.StateContextFlag
	}{
		{0},
		{icmodule.SCFlagNoHasPublicKeyInGetPRepTerm},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		t.Run(name, func(t *testing.T){
			sc := NewStateContextByFlags(nil, es, arg.flags)
			assert.Equal(t, arg.flags, sc.Flags())
		})
	}
}
