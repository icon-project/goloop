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
)

func TestJailInfo_IsEmpty(t *testing.T) {
	args := []struct {
		ji    JailInfo
		empty bool
	}{
		{JailInfo{0, 0, 0}, true},
		{JailInfo{1, 0, 0}, false},
		{JailInfo{0, 100, 0}, false},
		{JailInfo{0, 0, 100}, false},
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
			input{0, icmodule.PenaltyBlockValidation},
			output{JFlagInJail, true},
		},
		{
			input{JFlagInJail|JFlagDoubleVote, icmodule.PenaltyBlockValidation},
			output{JFlagInJail|JFlagDoubleVote, true},
		},
		{
			input{0, icmodule.PenaltyDoubleVote},
			output{JFlagInJail|JFlagDoubleVote, true},
		},
		{
			input{JFlagInJail, icmodule.PenaltyDoubleVote},
			output{JFlagInJail|JFlagDoubleVote, true},
		},
		{
			input{JFlagInJail|JFlagUnjailing, icmodule.PenaltyDoubleVote},
			output{JFlagInJail|JFlagDoubleVote, true},
		},
		{
			input{0, icmodule.PenaltyContinuousBlockValidation},
			output{0, false},
		},
		{
			input{JFlagInJail, icmodule.PenaltyContinuousBlockValidation},
			output{JFlagInJail, false},
		},
		{
			input{0, icmodule.PenaltyPRepDisqualification},
			output{0, false},
		},
		{
			input{JFlagInJail, icmodule.PenaltyPRepDisqualification},
			output{JFlagInJail, false},
		},
	}

	unjailRequestHeight := int64(500)
	sc := NewStateContext(1000, icmodule.RevisionIISS4, icmodule.RevisionIISS4)
	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			in := arg.in
			out := arg.out
			ji := &JailInfo{in.flags, unjailRequestHeight, 0}

			err := ji.OnPenaltyImposed(sc, in.pt)
			if out.success {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, out.flags, ji.Flags())
			assert.Equal(t, unjailRequestHeight, ji.UnjailRequestHeight())
			assert.Zero(t, ji.MinDoubleVoteHeight())
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
			&JailInfo{0, 0, 0}, 100,
			true, false, 0,
		},
		{
			&JailInfo{JFlagInJail, 100, 0}, 50,
			false, true, 100,
		},
		{
			&JailInfo{JFlagInJail, 0, 0}, 100,
			true, true, 100,
		},
		{
			&JailInfo{JFlagInJail | JFlagUnjailing, 50, 0}, 100,
			true, true, 50,
		},
		{
			&JailInfo{JFlagInJail | JFlagDoubleVote, 0, 0}, 100,
			true, true, 100,
		},
		{
			&JailInfo{JFlagInJail | JFlagUnjailing | JFlagDoubleVote, 50, 0}, 100,
			true, true, 50,
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			ji := arg.ji
			sc := NewStateContext(arg.bh, icmodule.RevisionIISS4, icmodule.RevisionIISS4)
			err := ji.OnUnjailRequested(sc)
			if arg.success {
				assert.NoError(t, err)
				assert.Equal(t, arg.inJail, ji.IsInJail())
				assert.Equal(t, arg.unjailRequestBlockHeight, ji.UnjailRequestHeight())
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
		mdvbh   int64 // MinDoubleVoteHeight
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
			JailInfo{JFlagInJail | JFlagDoubleVote, 50, 80},
			100,
			output{false, 0, 0},
		},
		{
			JailInfo{JFlagInJail | JFlagUnjailing, 50, 0},
			100,
			output{true, 50, 0},
		},
		{
			JailInfo{JFlagInJail | JFlagUnjailing | JFlagDoubleVote, 50, 0},
			100,
			output{true, 50, 100},
		},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			exp := arg.exp
			ji := arg.ji
			sc := NewStateContext(arg.bh, icmodule.RevisionPreIISS4, icmodule.RevisionIISS4)
			err := ji.OnMainPRepIn(sc, nil)
			if exp.success {
				assert.NoError(t, err)
				assert.Zero(t, ji.Flags())
				assert.Equal(t, exp.urbh, ji.UnjailRequestHeight())
				assert.Equal(t, exp.mdvbh, ji.MinDoubleVoteHeight())
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
			assert.Equal(t, arg.minDoubleVoteHeight, ji.minDoubleVoteHeight)
		})
	}
}

func TestJailInfo_Format(t *testing.T) {
	const (
		flags               = JFlagInJail
		unjailRequestHeight = 100
		minDoubleVoteHeight = 200
	)
	ji := JailInfo{flags, unjailRequestHeight, minDoubleVoteHeight}

	args := []struct {
		fmt    string
		output string
	}{
		{"%s", "JailInfo{1 100 200}"},
		{"%v", "JailInfo{1 100 200}"},
		{"%+v", "JailInfo{flags:1 urbh:100 mdvbh:200}"},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			text := fmt.Sprintf(arg.fmt, ji)
			assert.Equal(t, arg.output, text)
		})
	}
}
