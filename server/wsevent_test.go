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

package server

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

func stringPtr(s string) *string {
	return &s
}

func TestEventFilter_Compile(t *testing.T) {
	type fields struct {
		Addr      *common.Address
		Signature string
		Indexed   []*string
		Data      []*string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"Empty", fields{}, true},
		{"InvalidSig1", fields{Signature: "TestEvent"}, true},
		{"InvalidSig2", fields{Signature: "TestEvent(puha)"}, true},
		{"InvalidData1", fields{
			Addr:      nil,
			Signature: "TestEvent(Address)",
			Indexed:   []*string{stringPtr("hx")},
			Data:      nil,
		}, true},
		{"InvalidData2", fields{
			Addr:      nil,
			Signature: "TestEvent(int)",
			Data:      []*string{stringPtr("abcd")},
		}, true},
		{"InvalidData3", fields{
			Addr:      nil,
			Signature: "TestEvent(int)",
			Indexed:   []*string{stringPtr("0x1"), stringPtr("0x2")},
			Data:      nil,
		}, true},
		{"InvalidData4", fields{
			Addr:      nil,
			Signature: "TestEvent(int)",
			Indexed:   nil,
			Data:      []*string{stringPtr("0x1"), stringPtr("0x2")},
		}, true},
		{"Valid1", fields{
			Addr:      common.MustNewAddressFromString("cx02"),
			Signature: "TestEvent(Address,str,int,bool)",
			Indexed: []*string{
				stringPtr("hx1230000000000000000000000000000000000000"),
				stringPtr("hs'#F"),
			},
			Data: []*string{
				stringPtr("0x3a3"),
				stringPtr("0x1"),
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &EventFilter{
				Addr:      tt.fields.Addr,
				Signature: tt.fields.Signature,
				Indexed:   tt.fields.Indexed,
				Data:      tt.fields.Data,
			}
			if err := f.Compile(); (err != nil) != tt.wantErr {
				t.Errorf("Compile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testEventLog struct {
	module.EventLog
	addr       *common.Address
	indexed    [][]byte
	data       [][]byte
	indexedStr []*string
	dataStr    []*string
}

func (t *testEventLog) Address() module.Address {
	return t.addr
}

func (t *testEventLog) Indexed() [][]byte {
	return t.indexed
}

func (t *testEventLog) Data() [][]byte {
	return t.data
}

func (t *testEventLog) MarshalJSON() ([]byte, error) {
	obj := map[string]interface{}{
		"indexed": t.indexedStr,
	}
	if t.addr != nil {
		obj["scoreAddress"] = t.addr
	}
	if t.dataStr != nil {
		obj["data"] = t.dataStr
	}
	return json.Marshal(obj)
}

func newTestEventLog(addr string, signature string, indexed, data [][]string) *testEventLog {
	el := new(testEventLog)
	if len(addr) > 0 {
		el.addr = common.MustNewAddressFromString(addr)
	}
	el.indexed = append(el.indexed, []byte(signature))
	el.indexedStr = append(el.indexedStr, &signature)
	for _, typeAndValue := range indexed {
		if typeAndValue == nil {
			el.indexed = append(el.indexed, nil)
			el.indexedStr = append(el.indexedStr, nil)
			continue
		}
		if bs, err := txresult.EventDataStringToBytesByType(typeAndValue[0], typeAndValue[1]); err != nil {
			panic(err)
		} else {
			el.indexed = append(el.indexed, bs)
			el.indexedStr = append(el.indexedStr, &typeAndValue[1])
		}
	}
	for _, typeAndValue := range data {
		if typeAndValue == nil {
			el.data = append(el.data, nil)
			el.dataStr = append(el.dataStr, nil)
			continue
		}
		if bs, err := txresult.EventDataStringToBytesByType(typeAndValue[0], typeAndValue[1]); err != nil {
			panic(err)
		} else {
			el.data = append(el.data, bs)
			el.dataStr = append(el.dataStr, &typeAndValue[1])
		}
	}
	return el
}

func TestEventFilter_MatchLog(t *testing.T) {
	type fields struct {
		Addr      *common.Address
		Signature string
		Indexed   []*string
		Data      []*string
	}
	type args struct {
		fields fields
		want   bool
	}
	tests := []struct {
		name string
		log  module.EventLog
		args []args
	}{
		{
			"NoParamEvent",
			newTestEventLog("cx01", "NoParamEvent()", nil, nil),
			[]args{
				{
					fields{
						Addr:      common.MustNewAddressFromString("cx01"),
						Signature: "NoParamEvent()",
					},
					true,
				},
				{
					fields{
						Signature: "NoParamEvent()",
					},
					true,
				},
				{
					fields{
						Addr:      common.MustNewAddressFromString("cx02"),
						Signature: "NoParamEvent()",
					},
					false,
				},
				{
					fields{
						Addr:      common.MustNewAddressFromString("cx01"),
						Signature: "NoParamEvent(int)",
						Indexed:   []*string{stringPtr("0x1")},
					},
					false,
				},
				{
					fields{
						Addr:      common.MustNewAddressFromString("cx01"),
						Signature: "NoParamEvent(int)",
						Data:      []*string{stringPtr("0x1")},
					},
					false,
				},
			},
		},
		{
			"MultiParamEvent",
			newTestEventLog("cx03", "Parameter(int,str,bytes,bool)",
				[][]string{{"int", "-0x195"}, {"str", "puha"}, {"bytes", "0xab87"}},
				[][]string{{"bool", "0x1"}}),
			[]args{
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), nil, stringPtr("0xab87")},
					},
					true,
				},
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Data:      []*string{stringPtr("-0x195"), nil, stringPtr("0xab87")},
					},
					false,
				},
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), nil, stringPtr("0xab87"), nil},
					},
					false,
				},
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), nil, stringPtr("0xab87")},
						Data:      []*string{stringPtr("0x1")},
					},
					true,
				},
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), nil, stringPtr("0xab87")},
						Data:      []*string{stringPtr("0x0")},
					},
					false,
				},
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), stringPtr(""), stringPtr("0xab87")},
						Data:      []*string{stringPtr("0x1")},
					},
					false,
				},
			},
		},
		{
			"PartParamEvents",
			newTestEventLog("cx03", "Parameter(int,str,bytes,bool)",
				[][]string{{"int", "-0x195"}, {"str", "puha"}, nil},
				[][]string{{"bool", "0x1"}}),
			[]args{
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), stringPtr("puha")},
					},
					true,
				},
				{
					fields{
						Signature: "Parameter(int,str,bytes,bool)",
						Indexed:   []*string{stringPtr("-0x195"), nil, stringPtr("0xab87")},
					},
					false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for idx, arg := range tt.args {
				t.Run(fmt.Sprint(idx), func(t *testing.T) {
					f := &EventFilter{
						Addr:      arg.fields.Addr,
						Signature: arg.fields.Signature,
						Indexed:   arg.fields.Indexed,
						Data:      arg.fields.Data,
					}
					err := f.Compile()
					assert.NoError(t, err)
					if got := f.MatchLog(tt.log); got != arg.want {
						t.Errorf("MatchLog() = %v, want %v", got, arg.want)
					}
				})
			}
		})
	}
}

type testReceipt struct {
	module.Receipt
	events []*testEventLog
	lb     *txresult.LogsBloom
}

type testEventLogIterator struct {
	events []*testEventLog
	idx    int
}

func (itr *testEventLogIterator) Has() bool {
	return itr.idx < len(itr.events)
}

func (itr *testEventLogIterator) Next() error {
	if itr.idx < len(itr.events) {
		itr.idx += 1
		return nil
	} else {
		return errors.ErrInvalidState
	}
}

func (itr *testEventLogIterator) Get() (module.EventLog, error) {
	if itr.idx < 0 || itr.idx >= len(itr.events) {
		return nil, errors.ErrInvalidState
	}
	return itr.events[itr.idx], nil
}

func (t *testReceipt) LogsBloom() module.LogsBloom {
	return t.lb
}

func (t *testReceipt) EventLogIterator() module.EventLogIterator {
	return &testEventLogIterator{
		events: t.events,
	}
}

func newTestReceipt(events []*testEventLog) module.Receipt {
	lb := txresult.NewLogsBloom(nil)
	for _, event := range events {
		lb.AddAddressOfLog(event.Address())
		for idx, data := range event.Indexed() {
			if data == nil {
				continue
			}
			lb.AddIndexedOfLog(idx, data)
		}
	}
	return &testReceipt{
		events: events,
		lb:     lb,
	}
}

func TestEventFilter_MatchEvents(t *testing.T) {
	type fields struct {
		Addr      *common.Address
		Signature string
		Indexed   []*string
		Data      []*string
	}
	type args struct {
		fields fields
		want   []int
	}
	tests := []struct {
		name   string
		events []*testEventLog
		args   []args
	}{
		{
			"EmptyEvents",
			[]*testEventLog{},
			[]args{
				{
					fields{
						Signature: "TestEvent()",
					},
					[]int{},
				},
			},
		},
		{
			"SomeEvents",
			[]*testEventLog{
				newTestEventLog("cx01", "TestEventA()", nil, nil),
				newTestEventLog("cx02", "TestEventB(int,bytes)", [][]string{{"int", "0x1"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx03", "TestEventB(int,bytes)", [][]string{{"int", "0x2"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx02", "TestEventB(int,bytes)", [][]string{{"int", "0x3"}}, [][]string{nil}),
				newTestEventLog("cx02", "TestEventB(int,bytes)", [][]string{{"int", "0x3"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx03", "TestEventB(int,bytes)", [][]string{{"int", "0x3"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx03", "TestEventB(int,bytes)", [][]string{{"int", "0x4"}}, [][]string{nil}),
			},
			[]args{
				{
					fields{
						Signature: "TestEventC()",
					},
					[]int{},
				},
				{
					fields{
						Signature: "TestEventA()",
					},
					[]int{0},
				},
				{
					fields{
						Signature: "TestEventB(int,bytes)",
					},
					[]int{1, 2, 3, 4, 5, 6},
				},
				{
					fields{
						Signature: "TestEventB(int,bytes)",
						Indexed:   []*string{stringPtr("0x3")},
					},
					[]int{3, 4, 5},
				},
				{
					fields{
						Addr:      common.MustNewAddressFromString("cx02"),
						Signature: "TestEventB(int,bytes)",
					},
					[]int{1, 3, 4},
				},
				{
					fields{
						Signature: "TestEventB(int,bytes)",
						Data:      []*string{stringPtr("0x12ab")},
					},
					[]int{1, 2, 4, 5},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rct := newTestReceipt(tt.events)
			for idx, arg := range tt.args {
				t.Run(fmt.Sprint(idx), func(t *testing.T) {
					f := &EventFilter{
						Addr:      arg.fields.Addr,
						Signature: arg.fields.Signature,
						Indexed:   arg.fields.Indexed,
						Data:      arg.fields.Data,
					}
					err := f.Compile()
					assert.NoError(t, err)
					got1, got2, err := f.MatchEvents(rct, true)
					assert.NoError(t, err)

					assert.Equal(t, len(arg.want), len(got1))
					assert.Equal(t, len(arg.want), len(got2))
					for idx, value := range arg.want {
						assert.Equal(t, value, int(got1[idx].Value))
						assert.Equal(t, tt.events[value], got2[idx])
					}
				})
			}
		})
	}
}

func TestEventFilters_MatchEvents(t *testing.T) {
	type args struct {
		name    string
		filters EventFilters
		want    []int
	}
	tests := []struct {
		name   string
		events []*testEventLog
		args   []args
	}{
		{
			"EmptyEvents",
			[]*testEventLog{},
			[]args{
				{
					"Single",
					EventFilters{
						&EventFilter{
							Signature: "TestEvent()",
						},
					},
					[]int{},
				},
				{
					"NoFilter",
					EventFilters{},
					[]int{},
				},
				{
					"NilFilters",
					nil,
					[]int{},
				},
			},
		},
		{
			"SomeEvents",
			[]*testEventLog{
				newTestEventLog("cx01", "TestEventA()", nil, nil),
				newTestEventLog("cx02", "TestEventB(int,bytes)", [][]string{{"int", "0x1"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx03", "TestEventB(int,bytes)", [][]string{{"int", "0x2"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx02", "TestEventB(int,bytes)", [][]string{{"int", "0x3"}}, [][]string{nil}),
				newTestEventLog("cx02", "TestEventB(int,bytes)", [][]string{{"int", "0x3"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx03", "TestEventB(int,bytes)", [][]string{{"int", "0x3"}}, [][]string{{"bytes", "0x12ab"}}),
				newTestEventLog("cx03", "TestEventB(int,bytes)", [][]string{{"int", "0x4"}}, [][]string{nil}),
			},
			[]args{
				{
					"NilFilters",
					nil,
					[]int{},
				},
				{
					"Filter1",
					EventFilters{
						{
							Signature: "TestEventB(int,bytes)",
							Indexed:   []*string{stringPtr("0x3")},
						},
					},
					[]int{3, 4, 5},
				},
				{
					"Filter2",
					EventFilters{
						{
							Addr:      common.MustNewAddressFromString("cx02"),
							Signature: "TestEventB(int,bytes)",
						},
					},
					[]int{1, 3, 4},
				},
				{
					"Filter1Or2",
					EventFilters{
						{
							Signature: "TestEventB(int,bytes)",
							Indexed:   []*string{stringPtr("0x3")},
						},
						{
							Addr:      common.MustNewAddressFromString("cx02"),
							Signature: "TestEventB(int,bytes)",
						},
					},
					[]int{1, 3, 4, 5},
				},
				{
					"Filter1And2",
					EventFilters{
						{
							Addr:      common.MustNewAddressFromString("cx02"),
							Signature: "TestEventB(int,bytes)",
							Indexed:   []*string{stringPtr("0x3")},
						},
					},
					[]int{3, 4},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rct := newTestReceipt(tt.events)
			for _, arg := range tt.args {
				t.Run(arg.name, func(t *testing.T) {
					for _, filter := range arg.filters {
						err := filter.Compile()
						assert.NoError(t, err)
					}
					got1, got2, err := arg.filters.MatchEvents(rct, true)
					assert.NoError(t, err)

					assert.Equal(t, len(arg.want), len(got1))
					assert.Equal(t, len(arg.want), len(got2))
					for idx, value := range arg.want {
						assert.Equal(t, value, int(got1[idx].Value))
						assert.Equal(t, tt.events[value], got2[idx])
					}
				})
			}
		})
	}
}

func TestEventRequest_Compile(t *testing.T) {
	type fields struct {
		EventFilter EventFilter
		Height      common.HexInt64
		Logs        common.HexBool
		Filters     EventFilters
	}
	tests := []struct {
		name    string
		fields  fields
		want    EventFilters
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"LegacySuccess",
			fields{
				EventFilter: EventFilter{
					Signature: "TestEvent()",
				},
			},
			EventFilters{
				&EventFilter{
					Signature: "TestEvent()",
				},
			},
			assert.NoError,
		},
		{
			"LegacyFail",
			fields{
				EventFilter: EventFilter{
					Signature: "TestEvent(",
				},
			},
			nil,
			assert.Error,
		},
		{
			"MultiAndSingle",
			fields{
				EventFilter: EventFilter{
					Signature: "TestEvent(",
				},
				Filters: EventFilters{
					&EventFilter{
						Signature: "TestEvent2()",
					},
				},
			},
			nil,
			assert.Error,
		},
		{
			"MultiFilters",
			fields{
				Filters: EventFilters{
					&EventFilter{
						Signature: "TestEvent2()",
					},
				},
			},
			EventFilters{
				&EventFilter{
					Signature: "TestEvent2()",
				},
			},
			assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &EventRequest{
				EventFilter: tt.fields.EventFilter,
				Height:      tt.fields.Height,
				Logs:        tt.fields.Logs,
				Filters:     tt.fields.Filters,
			}
			got, err := f.Compile()
			if !tt.wantErr(t, err, fmt.Sprintf("Compile()")) {
				return
			}
			for _, filter := range tt.want {
				filter.Compile()
			}
			assert.EqualValuesf(t, tt.want, got, "Compile()")
		})
	}
}
