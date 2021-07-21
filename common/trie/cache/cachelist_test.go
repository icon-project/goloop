/*
 * Copyright 2021 ICON Foundation
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

package cache

import (
	"reflect"
	"testing"
)

func TestNewNodeCacheList(t *testing.T) {

}

func dummyFactory(id string) *NodeCache {
	return new(NodeCache)
}

func TestNewNodeCacheList1(t *testing.T) {
	type args struct {
		sample   int
		limit    int
		scenario []string
	}
	tests := []struct {
		name string
		args args
		want []bool
	}{
		{
			"One", args{10, 1,
				[]string{"1"}},
			[]bool{true},
		},
		{
			"HitSames", args{4, 2,
				[]string{"1", "2", "3", "4"}},
			[]bool{true, true, false, false},
		},
		{
			"PushOut", args{4, 2,
				[]string{"1", "2", "3", "4", "5", "6"}},
			[]bool{true, true, false, false, true, true},
		},
		{
			"HitMore1", args{6, 2,
				[]string{"1", "2", "3", "4", "3", "4", "2"}},
			[]bool{true, true, false, false, true, true, false},
		},
		{
			"HitMore2", args{6, 2,
				[]string{"1", "2", "3", "4", "5", "6", "3", "4"}},
			[]bool{true, true, false, false, false, false, true, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewNodeCacheList(tt.args.sample, tt.args.limit, dummyFactory)
			var got []bool
			for _, s := range tt.args.scenario {
				got = append(got, cache.Get(s) != nil)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNodeCacheList() = %v, want %v", got, tt.want)
			}
		})
	}
}
