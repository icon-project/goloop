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

package jsonrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name string
		c    ErrorCode
		want string
	}{
		{"ServerError", ErrorCodeServer, "ServerError"},
		{"ServerError(001)", ErrorCodeServer - 1, "ServerError(-32001)"},
		{"ServerError(999)", ErrorCodeServer - 999, "ServerError(-32999)"},
		{"SystemError", ErrorCodeSystem, "SystemError"},
		{"SystemError(010)", ErrorCodeSystem - 10, "SystemError(-31010)"},
		{"SystemError(999)", ErrorCodeSystem - 999, "SystemError(-31999)"},
		{"SCOREError(0)", ErrorCodeScore, "SCOREError(-30000)"},
		{"SCOREError(1)", ErrorCodeScore - 1, "SCOREError(-30001)"},
		{"SCOREError(999)", ErrorCodeScore - 999, "SCOREError(-30999)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.c.String(), "String()")
		})
	}
}
