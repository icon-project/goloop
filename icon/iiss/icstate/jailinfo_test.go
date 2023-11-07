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
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/icon/icmodule"
	"github.com/icon-project/goloop/icon/iiss/icutils"
)

func checkUnjailRequestHeight(t *testing.T, ji *JailInfo, unjailRequestHeight int64) {
	if !ji.IsUnjailing() {
		unjailRequestHeight = int64(0)
	}
	assert.Equal(t, unjailRequestHeight, ji.UnjailRequestHeight())
}

func newJailInfo(flags int, unjailRequestHeight, minDoubleSignHeight int64) *JailInfo {
	if !icutils.MatchAll(flags, JFlagUnjailing) && unjailRequestHeight != 0 {
		return nil
	}
	if icutils.MatchAny(flags, ^(JFlagMax - 1)) {
		return nil
	}
	return &JailInfo{flags, unjailRequestHeight, minDoubleSignHeight}
}

func TestJailInfo_IsEmpty(t *testing.T) {
	args := []struct {
		ji    *JailInfo
		empty bool
	}{
		{newJailInfo(0, 0, 0), true},
		{newJailInfo(JFlagInJail, 0, 0), false},
		{newJailInfo(JFlagInJail|JFlagUnjailing, 50, 0), false},
		{newJailInfo(JFlagInJail|JFlagUnjailing|JFlagDoubleSign, 50, 0), false},
		{newJailInfo(0, 0, 100), false},
		{newJailInfo(JFlagInJail|JFlagDoubleSign, 0, 100), false},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.empty, arg.ji.IsEmpty())
		})
	}
}

func TestJailInfo_OnPenaltyImposed(t *testing.T) {
	type input struct {
		flags int
		pt    icmodule.PenaltyType
	}
	type output struct {
		flags   int
		success bool
	}
	args := []struct {
		in  input
		out output
	}{
		{
			input{0, icmodule.PenaltyValidationFailure},
			output{JFlagInJail, true},
		},
		{
			input{JFlagInJail | JFlagDoubleSign, icmodule.PenaltyValidationFailure},
			output{JFlagInJail | JFlagDoubleSign, true},
		},
		{
			input{0, icmodule.PenaltyDoubleSign},
			output{JFlagInJail | JFlagDoubleSign, true},
		},
		{
			input{JFlagInJail, icmodule.PenaltyDoubleSign},
			output{JFlagInJail | JFlagDoubleSign, true},
		},
		{
			input{JFlagInJail | JFlagAccumulatedValidationFailure, icmodule.PenaltyDoubleSign},
			output{JFlagInJail | JFlagAccumulatedValidationFailure | JFlagDoubleSign, true},
		},
		{
			input{JFlagInJail | JFlagUnjailing, icmodule.PenaltyDoubleSign},
			output{JFlagInJail | JFlagDoubleSign, true},
		},
		{
			input{0, icmodule.PenaltyAccumulatedValidationFailure},
			output{JFlagInJail | JFlagAccumulatedValidationFailure, true},
		},
		{
			input{JFlagInJail, icmodule.PenaltyAccumulatedValidationFailure},
			output{JFlagInJail | JFlagAccumulatedValidationFailure, true},
		},
		{
			input{JFlagInJail | JFlagDoubleSign, icmodule.PenaltyAccumulatedValidationFailure},
			output{JFlagInJail | JFlagAccumulatedValidationFailure | JFlagDoubleSign, true},
		},
		{
			input{
				JFlagInJail | JFlagDoubleSign | JFlagUnjailing,
				icmodule.PenaltyAccumulatedValidationFailure,
			},
			output{JFlagInJail | JFlagAccumulatedValidationFailure | JFlagDoubleSign, true},
		},
		{
			input{0, icmodule.PenaltyPRepDisqualification},
			output{0, false},
		},
		{
			input{JFlagInJail, icmodule.PenaltyPRepDisqualification},
			output{JFlagInJail, false},
		},
		{
			input{0, icmodule.PenaltyMissedNetworkProposalVote},
			output{0, false},
		},
		{
			input{JFlagInJail, icmodule.PenaltyMissedNetworkProposalVote},
			output{JFlagInJail, false},
		},
	}

	var unjailRequestHeight int64
	sc := newMockStateContext(map[string]interface{}{
		"blockHeight": int64(1000),
		"revision":    icmodule.RevisionIISS4R1,
	})
	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			in := arg.in
			out := arg.out
			if icutils.MatchAll(in.flags, JFlagUnjailing) {
				unjailRequestHeight = int64(500)
			} else {
				unjailRequestHeight = int64(0)
			}
			ji := newJailInfo(in.flags, unjailRequestHeight, 0)

			err := ji.OnPenaltyImposed(sc, in.pt)
			if out.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, out.flags, ji.Flags())
			checkUnjailRequestHeight(t, ji, unjailRequestHeight)
			assert.Zero(t, ji.MinDoubleSignHeight())
		})
	}
}

func TestJailInfo_OnUnjailRequested(t *testing.T) {
	args := []struct {
		// input
		ji *JailInfo
		bh int64
		// output
		success                  bool
		inJail                   bool
		unjailRequestBlockHeight int64
	}{
		{
			newJailInfo(0, 0, 0), 100,
			false, false, 0,
		},
		{
			newJailInfo(JFlagInJail|JFlagUnjailing, 50, 0), 100,
			false, true, 50,
		},
		{
			newJailInfo(JFlagInJail|JFlagUnjailing|JFlagDoubleSign, 50, 0), 100,
			false, true, 50,
		},
		{
			newJailInfo(JFlagInJail, 0, 0), 100,
			true, true, 100,
		},
		{
			newJailInfo(JFlagInJail|JFlagDoubleSign, 0, 0), 100,
			true, true, 100,
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			ji := arg.ji
			sc := newMockStateContext(map[string]interface{}{
				"blockHeight": arg.bh,
				"revision":    icmodule.RevisionIISS4R1,
			})
			err := ji.OnUnjailRequested(sc)
			if arg.success {
				assert.NoError(t, err)
				assert.Equal(t, arg.inJail, ji.IsInJail())
				checkUnjailRequestHeight(t, ji, arg.unjailRequestBlockHeight)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestJailInfo_OnMainPRepIn(t *testing.T) {
	type output struct {
		success bool
		urbh    int64 // UnjailRequestHeight
		mdsbh   int64 // MinDoubleSignHeight
	}
	args := []struct {
		// input
		ji JailInfo
		bh int64
		// output
		exp output
	}{
		{
			JailInfo{0, 0, 0},
			100,
			output{true, 0, 0},
		},
		{
			JailInfo{JFlagInJail, 0, 0},
			100,
			output{false, 0, 0},
		},
		{
			JailInfo{JFlagInJail | JFlagDoubleSign, 50, 80},
			100,
			output{false, 0, 0},
		},
		{
			JailInfo{JFlagInJail | JFlagUnjailing, 50, 0},
			100,
			output{true, 50, 0},
		},
		{
			JailInfo{JFlagInJail | JFlagUnjailing | JFlagDoubleSign, 50, 0},
			100,
			output{true, 50, 100},
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			exp := arg.exp
			ji := arg.ji
			sc := newMockStateContext(map[string]interface{}{
				"blockHeight": arg.bh,
				"revision":    icmodule.RevisionIISS4R1,
			})
			err := ji.OnMainPRepIn(sc)
			if exp.success {
				assert.NoError(t, err)
				assert.Zero(t, ji.Flags())
				checkUnjailRequestHeight(t, &ji, exp.urbh)
				assert.Equal(t, exp.mdsbh, ji.MinDoubleSignHeight())
			} else {
				assert.Error(t, err)
				assert.Equal(t, arg.ji, ji)
			}
		})
	}
}

func TestJailInfo_RLPEncodeSelf(t *testing.T) {
	var err error
	args := []JailInfo{
		{0, 0, 0},
		{JFlagInJail, 0, 0},
		{0, 100, 0},
		{0, 0, 100},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			e := codec.BC.NewEncoder(buf)
			err = arg.RLPEncodeSelf(e)
			assert.NoError(t, err)
			err = e.Close()
			assert.NoError(t, err)

			ji := JailInfo{}
			assert.True(t, ji.IsEmpty())

			d := codec.BC.NewDecoder(bytes.NewBuffer(buf.Bytes()))
			err = ji.RLPDecodeSelf(d)
			assert.Equal(t, arg.flags, ji.flags)
			assert.Equal(t, arg.unjailRequestHeight, ji.unjailRequestHeight)
			assert.Equal(t, arg.minDoubleSignHeight, ji.minDoubleSignHeight)
		})
	}
}

func TestJailInfo_Format(t *testing.T) {
	const (
		flags               = JFlagInJail | JFlagUnjailing
		unjailRequestHeight = 100
		minDoubleSignHeight = 200
	)
	ji := newJailInfo(flags, unjailRequestHeight, minDoubleSignHeight)

	args := []struct {
		fmt    string
		output string
	}{
		{"%s", "JailInfo{3 100 200}"},
		{"%v", "JailInfo{3 100 200}"},
		{"%+v", "JailInfo{flags:3 urbh:100 mdsbh:200}"},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			text := fmt.Sprintf(arg.fmt, ji)
			assert.Equal(t, arg.output, text)
		})
	}
}

func TestJailInfo_IsFunctions(t *testing.T) {
	type output struct {
		unjailable          bool
		electable           bool
		inJail              bool
		unjailing           bool
		inDoubleSignPenalty bool
	}
	args := []struct {
		// input
		flags int
		urbh  int64
		mdsbh int64
		// output
		out output
	}{
		{
			JFlagInJail, 0, 0,
			output{true, false, true, false, false},
		},
		{
			JFlagInJail, 0, 200,
			output{true, false, true, false, false},
		},
		{
			JFlagInJail | JFlagDoubleSign, 0, 0,
			output{true, false, true, false, true},
		},
		{
			JFlagInJail | JFlagDoubleSign, 0, 200,
			output{true, false, true, false, true},
		},
		{
			0, 0, 0,
			output{false, true, false, false, false},
		},
		{
			JFlagInJail | JFlagUnjailing, 100, 0,
			output{false, true, true, true, false},
		},
		{
			JFlagInJail | JFlagUnjailing, 100, 200,
			output{false, true, true, true, false},
		},
		{
			JFlagInJail | JFlagUnjailing | JFlagDoubleSign, 100, 0,
			output{false, true, true, true, true},
		},
		{
			JFlagInJail | JFlagUnjailing | JFlagDoubleSign, 100, 200,
			output{false, true, true, true, true},
		},
	}
	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			ji := newJailInfo(arg.flags, arg.urbh, arg.mdsbh)
			assert.NotNil(t, ji)
			assert.Equal(t, arg.out.unjailable, ji.IsUnjailable())
			assert.Equal(t, arg.out.electable, ji.IsElectable())
			assert.Equal(t, arg.out.inJail, ji.IsInJail())
			assert.Equal(t, arg.out.unjailing, ji.IsUnjailing())
			assert.Equal(t, arg.out.inDoubleSignPenalty, icutils.MatchAll(ji.Flags(), JFlagDoubleSign))
		})
	}
}
