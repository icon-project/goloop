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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
)

func TestBlockRequest_Compile(t *testing.T) {
	type fields struct {
		Height       common.HexInt64
		EventFilters []*EventFilter
		Logs         common.HexBool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"NoFilters",
			fields{},
			assert.NoError,
		},
		{
			"MultiFilters",
			fields{
				EventFilters: []*EventFilter{
					{
						Signature: "EventNoParam()",
					},
					{
						Signature: "EventParam(int)",
					},
				},
			},
			assert.NoError,
		},
		{
			"WithNilFilters",
			fields{
				EventFilters: []*EventFilter{
					{
						Signature: "EventNoParam()",
					},
					nil,
					{
						Signature: "EventParam(int)",
					},
				},
			},
			assert.Error,
		},
		{
			"WithInvalidFilter",
			fields{
				EventFilters: []*EventFilter{
					{
						Signature: "EventNoParam()",
					},
					{
						Signature: "EventParam(int",
					},
				},
			},
			assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BlockRequest{
				Height:       tt.fields.Height,
				EventFilters: tt.fields.EventFilters,
				Logs:         tt.fields.Logs,
			}
			tt.wantErr(t, r.Compile(), fmt.Sprintf("Compile()"))
		})
	}
}

func TestWsSessionManager_RunBlockSession(t *testing.T) {
	logger := log.New()
	logger.SetOutput(io.Discard)

	t1 := make(chan string, 1)
	c1 := make(chan string, 1)
	upgrader := newTestWebsocketUpgrader(func(ctx echo.Context, conn *testWebSocketConn) {
		t1 <- "NEW"
		go func() {
			assert.Equal(t, "REQUEST", <-c1)
			err := conn.clientWriteJSON(map[string]interface{}{
				"height": "0x1",
				"eventFilters": []interface{}{
					map[string]interface{}{
						"event": "EventLog1()",
					},
					map[string]interface{}{
						"event": "EventLog2()",
					},
				},
			})
			assert.NoError(t, err)

			bs, err := conn.clientRead()
			assert.NoError(t, err)

			var res WSResponse
			err = json.Unmarshal(bs, &res)
			assert.NoError(t, err)
			t1 <- fmt.Sprint("RESULT:", res.Code)

			go func() {
				for value := <-c1; value == "WAIT"; value = <-c1 {
					t.Logf("[%p] WAIT notification", conn)
					if bs, err := conn.clientRead(); err != nil {
						t.Logf("[%p] STOP msg=err", conn)
						break
					} else {
						t.Logf("[%p] RECEIVED msg=%s\n", conn, bs)
						t1 <- string(bs)
					}
				}
				conn.Close()
				t1 <- "CLOSED"
			}()
		}()
	})
	wm := newWSSessionManagerWithUpgrader(logger, 10, upgrader)

	s1 := make(chan string, 1)
	blkReceipts := blockReceipts{
		"empty": testReceiptList{},
		"1": testReceiptList{
			newTestReceipt([]*testEventLog{
				newTestEventLog("cx01", "EventLog1()", nil, nil),
			}),
		},
		"2": testReceiptList{
			newTestReceipt([]*testEventLog{
				newTestEventLog("cx01", "EventLog1()", nil, nil),
			}),
			newTestReceipt([]*testEventLog{
				newTestEventLog("cx02", "EventLog1()", nil, nil),
				newTestEventLog("cx04", "EventLog2()", nil, nil),
				newTestEventLog("cx03", "EventLog1()", nil, nil),
			}),
		},
	}
	chain := newTestChain(0,
		func(h int64) (getBlockFunc, error) {
			t.Logf("GetBlock(%d)", h)
			if h < 2 {
				_, _ = <-s1
				return func() module.Block {
					return &testBlock{
						height: h,
						result: "empty",
					}
				}, nil
			} else if h < 3 {
				return func() module.Block {
					_, _ = <-s1
					return &testBlock{
						height: h,
						result: "1",
						lb:     blkReceipts["1"].LogsBloom(),
					}
				}, nil
			} else {
				return func() module.Block {
					_, _ = <-s1
					return &testBlock{
						height: h,
						result: "2",
						lb:     blkReceipts["2"].LogsBloom(),
					}
				}, nil
			}
		},
		blkReceipts,
	)
	go wm.RunBlockSession(newTestContext(chain))

	// wait for the client and send request
	assert.Equal(t, "NEW", <-t1)
	c1 <- "REQUEST"

	// wait for the result
	assert.Equal(t, "RESULT:0", <-t1)

	waitNotification := func() *BlockNotification {
		var noti *BlockNotification
		var msg string

		msg = <-t1
		noti = new(BlockNotification)
		err := json.Unmarshal([]byte(msg), noti)
		if err != nil {
			buf := bytes.NewBuffer(nil)
			json.Indent(buf, []byte(msg), "", "  ")
			t.Logf("Fail to parse err=%+v JSON=\n%s", err, buf.Bytes())
		}
		assert.NoError(t, err)
		return noti
	}

	// wait a message (empty)
	s1 <- "OK"
	c1 <- "WAIT"
	n1 := waitNotification()
	assert.Equal(t, &BlockNotification{
		Hash:   testHeightToBlockID(1),
		Height: common.HexInt64{Value: 1},
	}, n1)

	// wait a message (1)
	c1 <- "WAIT"
	s1 <- "OK"
	n2 := waitNotification()
	assert.Equal(t, &BlockNotification{
		Hash:   testHeightToBlockID(2),
		Height: common.HexInt64{Value: 2},
		Indexes: [][]common.HexInt32{
			{
				{0},
			},
			{},
		},
		Events: [][][]common.HexInt32{
			{
				{
					{0},
				},
			},
			{},
		},
	}, n2)

	// wait a message (2)
	s1 <- "OK"
	c1 <- "WAIT"
	n3 := waitNotification()
	assert.Equal(t, &BlockNotification{
		Hash:   testHeightToBlockID(3),
		Height: common.HexInt64{Value: 3},
		Indexes: [][]common.HexInt32{
			{
				{0}, {1},
			},
			{
				{1},
			},
		},
		Events: [][][]common.HexInt32{
			{
				{{0}}, {{0}, {2}},
			},
			{
				{{1}},
			},
		},
	}, n3)

	// client quit
	c1 <- "QUIT"

	// wait for CLOSED
	assert.Equal(t, "CLOSED", <-t1)
	close(s1)

	wm.StopAllSessions()
}
