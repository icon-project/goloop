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

package icmodule

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableStatus_IsFuncs(t *testing.T) {
	args := []struct {
		status               EnableStatus
		isEnabled            bool
		isDisableTemp        bool
		isDisablePermanently bool
		isJail               bool
		isUnjail             bool
	}{
		{ESEnable, true, false, false, false, false},
		{ESDisableTemp, false, true, false, false, false},
		{ESDisablePermanent, false, false, true, false, false},
		{ESJail, false, false, false, true, false},
		{ESUnjail, false, false, false, false, true},
		{ESEnableAtNextTerm, false, false, false, false, false},
	}
	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.isEnabled, arg.status.IsEnabled())
			assert.Equal(t, arg.isDisableTemp, arg.status.IsDisabledTemporarily())
			assert.Equal(t, arg.isDisablePermanently, arg.status.IsDisabledPermanently())
			assert.Equal(t, arg.isJail, arg.status.IsJail())
			assert.Equal(t, arg.isUnjail, arg.status.IsUnjail())
		})
	}
}

func TestEnableStatus_String(t *testing.T) {
	args := []struct {
		status EnableStatus
		text   string
	}{
		{ESEnable, "Enable"},
		{ESDisableTemp, "DisableTemporarily"},
		{ESDisablePermanent, "DisablePermanently"},
		{ESJail, "Jail"},
		{ESUnjail, "Unjail"},
		{ESEnableAtNextTerm, "EnableAtNextTerm"},
		{ESMax, "Unknown"},
	}
	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.text, arg.status.String())
		})
	}
}
