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

package icstate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPRepCountConfigValid(t *testing.T) {
	args := []struct {
		main, sub, extra int
		valid            bool
	}{
		{22, 78, 0, true},
		{22, 78, 3, true},
		{19, 81, 9, true},
		{0, 10, 0, false},
		{-22, 10, 0, false},
		{22, -10, 3, false},
		{22, 78, -3, false},
		{22, 9, 10, false},
		{22, 9, 9, true},
		{22, 9, 3, true},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		t.Run(name, func(t *testing.T) {
			cfg := NewPRepCountConfig(arg.main, arg.sub, arg.extra)
			assert.Equal(t, arg.valid, IsPRepCountConfigValid(cfg))
		})
	}
}

func TestNewPRepCountConfig(t *testing.T) {
	args := []struct {
		main, sub, extra int
	}{
		{22, 78, 0},
		{22, 78, 3},
		{19, 81, 9},
	}

	for i, arg := range args {
		name := fmt.Sprintf("case-%02d", i)
		t.Run(name, func(t *testing.T) {
			main := arg.main
			sub := arg.sub
			extra := arg.extra
			cfg := NewPRepCountConfig(arg.main, arg.sub, arg.extra)

			assert.Equal(t, main, cfg.MainPReps(MainPRepNormal))
			assert.Equal(t, sub, cfg.SubPReps())
			assert.Equal(t, extra, cfg.MainPReps(MainPRepExtra))
			assert.Equal(t, main+extra, cfg.MainPReps(MainPRepAll))
			assert.Equal(t, main+sub, cfg.ElectedPReps())
		})
	}
}
