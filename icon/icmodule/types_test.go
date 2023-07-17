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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRate_MulInt64(t *testing.T) {
	args := []struct{
		v int64
		r int64
		result int64
	}{
		{0, 0, 0},
		{100, 0, 0},
		{-100, 0, 0},
		{100, 5000, 50},
		{100, 2000, 20},
		{ 100, 100, 1},
		{ -100, 5000, -50},
		{ 100, 150, 1},
		{ -100, 150, -1},
		{ 1000, 150, 15},
		{ -1000, 150, -15},
		{1000, 10000, 1000},
		{ -1000, 10000, -1000},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T){
			rate := Rate(arg.r)
			assert.Equal(t, arg.result, rate.MulInt64(arg.v))
		})
	}
}

func TestRate_MulBigInt(t *testing.T) {
	args := []struct{
		v int64
		r int64
		result int64
	}{
		{0, 0, 0},
		{100, 0, 0},
		{-100, 0, 0},
		{100, 5000, 50},
		{100, 2000, 20},
		{ 100, 100, 1},
		{ -100, 5000, -50},
		{ 100, 150, 1},
		{ -100, 150, -1},
		{ 1000, 150, 15},
		{ -1000, 150, -15},
		{1000, 10000, 1000},
		{ -1000, 10000, -1000},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T){
			v := big.NewInt(arg.v)
			rate := Rate(arg.r)
			result := rate.MulBigInt(v)
			assert.Zero(t, big.NewInt(arg.result).Cmp(result))
		})
	}
}

func TestRate_String(t *testing.T) {
	args := []struct{
		r int64
		result string
	}{
		{0, "0.0"},
		{ 10, "0.001"},
		{ 100, "0.01"},
		{1000, "0.1"},
		{10000, "1.0"},
		{20000, "2.0"},
		{20001, "2.0001"},
		{ -10, "-0.001"},
		{ -100, "-0.01"},
		{-1000, "-0.1"},
		{-10000, "-1.0"},
		{-20000, "-2.0"},
		{-20001, "-2.0001"},
		{17, "0.0017"},
		{-17, "-0.0017"},
	}

	for _, arg := range args {
		name := fmt.Sprintf("name-(%d)", arg.r)
		t.Run(name, func(t *testing.T){
			rate := Rate(arg.r)
			decimals := fmt.Sprintf("%s", rate)
			assert.Equal(t, arg.result, decimals)
		})
	}
}

func TestRate_IsValid(t *testing.T) {
	args := []struct{
		br Rate
		valid bool
	}{
		{0, true},
		{200, true},
		{5000, true},
		{10000, true},
		{-1, false},
		{-200, false},
		{-5000, false},
		{-10000, false},
		{10001, false},
		{20000, false},
	}

	for i, arg := range args {
		name := fmt.Sprintf("name-%02d", i)
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, arg.valid, arg.br.IsValid())
		})
	}
}
