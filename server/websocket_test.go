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
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/txresult"
)

type testWebSocketHandler func(ctx echo.Context, conn *testWebSocketConn)

type testWebSocketUpgrader struct {
	WebSocketUpgrader
	handler testWebSocketHandler
}

func (u *testWebSocketUpgrader) Upgrade(ctx echo.Context) (WebSocketConn, error) {
	conn := &testWebSocketConn{
		in:  make(chan interface{}, 3),
		out: make(chan interface{}, 3),
	}
	if u.handler != nil {
		u.handler(ctx, conn)
	}
	return conn, nil
}

func newTestWebsocketUpgrader(handler testWebSocketHandler) *testWebSocketUpgrader {
	return &testWebSocketUpgrader{handler: handler}
}

type testWebSocketConn struct {
	WebSocketConn
	lock    sync.Mutex
	in, out chan interface{}
	closed  bool
}

func (c *testWebSocketConn) Close() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.closed {
		close(c.in)
		close(c.out)
		c.closed = true
		return nil
	} else {
		return io.ErrClosedPipe
	}
}

func (c *testWebSocketConn) clientWrite(bs []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return io.ErrClosedPipe
	}
	c.in <- bs
	return nil
}

func (c *testWebSocketConn) clientWriteJSON(obj interface{}) error {
	bs, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.clientWrite(bs)
}

func (c *testWebSocketConn) clientRead() ([]byte, error) {
	bs, ok := <-c.out
	if !ok {
		return nil, io.ErrClosedPipe
	}
	return bs.([]byte), nil
}

func (c *testWebSocketConn) WriteJSON(v interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closed {
		return io.EOF
	}
	bs, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.out <- bs
	return nil
}

func (c *testWebSocketConn) ReadMessage() (messageType int, p []byte, err error) {
	o, ok := <-c.in
	if !ok {
		return 0, nil, io.ErrClosedPipe
	}
	switch obj := o.(type) {
	case error:
		return 0, nil, obj
	case []byte:
		return 0, obj, nil
	default:
		panic("InvalidObject")
	}
}

func (c *testWebSocketConn) NextReader() (messageType int, r io.Reader, err error) {
	o, ok := <-c.in
	if !ok {
		return 0, nil, io.ErrClosedPipe
	}
	switch obj := o.(type) {
	case error:
		return 0, nil, obj
	case []byte:
		buf := bytes.NewBuffer(obj)
		return 0, buf, nil
	default:
		panic("InvalidObject")
	}
}

type testContext struct {
	echo.Context
	config map[string]interface{}
}

func (ctx *testContext) Get(key string) interface{} {
	return ctx.config[key]
}

func newTestContext(chain module.Chain) *testContext {
	return &testContext{
		config: map[string]interface{}{
			"chain": chain,
		},
	}
}

type testChain struct {
	module.Chain
	bm module.BlockManager
	sm module.ServiceManager
	gs module.GenesisStorage
}

func (c *testChain) BlockManager() module.BlockManager {
	return c.bm
}

func (c *testChain) ServiceManager() module.ServiceManager {
	return c.sm
}

func (c *testChain) GenesisStorage() module.GenesisStorage {
	return c.gs
}

type getBlockFunc func() module.Block
type blockFetcher func(h int64) (getBlockFunc, error)
type blockReceipts map[string]testReceiptList

type testReceiptList []module.Receipt
type testReceiptIterator struct {
	receipts []module.Receipt
	index    int
}

func (it *testReceiptIterator) Has() bool {
	return it.index < len(it.receipts)
}

func (it *testReceiptIterator) Next() error {
	if it.index < len(it.receipts) {
		it.index += 1
	}
	if it.index >= len(it.receipts) {
		return errors.ErrInvalidState
	}
	return nil
}

func (it *testReceiptIterator) Get() (module.Receipt, error) {
	if it.index < len(it.receipts) {
		return it.receipts[it.index], nil
	} else {
		return nil, errors.ErrInvalidState
	}
}

func (t testReceiptList) Get(i int) (module.Receipt, error) {
	if i < 0 || i >= len(t) {
		return nil, errors.ErrInvalidState
	}
	return t[i], nil
}

func (t testReceiptList) GetProof(n int) ([][]byte, error) {
	panic("implement me")
}

func (t testReceiptList) Iterator() module.ReceiptIterator {
	return &testReceiptIterator{
		receipts: t,
		index:    0,
	}
}

func (t testReceiptList) Hash() []byte {
	panic("implement me")
}

func (t testReceiptList) Flush() error {
	panic("implement me")
}

func (t testReceiptList) LogsBloom() module.LogsBloom {
	lb := txresult.NewLogsBloom(nil)
	for _, r := range t {
		lb.Merge(r.LogsBloom())
	}
	return lb
}

func newTestChain(height int64, fetcher blockFetcher, receipts blockReceipts) module.Chain {
	return &testChain{
		bm: &testBlockManager{fetcher: fetcher},
		sm: &testServiceManager{receipts: receipts},
		gs: &testGenesisStorage{height: height},
	}
}

type testBlockManager struct {
	module.BlockManager
	fetcher blockFetcher
}

type testBlock struct {
	module.Block
	height int64
	result string
	lb     module.LogsBloom
}

func testHeightToBlockID(height int64) []byte {
	id := make([]byte, 32)
	big.NewInt(height).FillBytes(id)
	return id
}

func (b *testBlock) ID() []byte {
	return testHeightToBlockID(b.height)
}

func (b *testBlock) Result() []byte {
	return []byte(b.result)
}

func (b *testBlock) LogsBloom() module.LogsBloom {
	if b.lb != nil {
		return b.lb
	} else {
		return txresult.NewLogsBloom(nil)
	}
}

func (bm *testBlockManager) WaitForBlock(h int64) (<-chan module.Block, error) {
	getBlock, err := bm.fetcher(h)
	if err != nil {
		return nil, err
	}
	ch := make(chan module.Block, 1)
	go func() {
		ch <- getBlock()
	}()
	return ch, nil
}

type testServiceManager struct {
	module.ServiceManager
	receipts blockReceipts
}

func (sm *testServiceManager) ReceiptListFromResult(result []byte, group module.TransactionGroup) (module.ReceiptList, error) {
	list, ok := sm.receipts[string(result)]
	if !ok {
		return nil, errors.ErrNotFound
	}
	return list, nil
}

type testGenesisStorage struct {
	module.GenesisStorage
	height int64
}

func (gs *testGenesisStorage) Height() int64 {
	return gs.height
}

func TestWSSessionManager_InvalidRequest(t *testing.T) {
	logger := log.New()
	logger.SetOutput(io.Discard)

	t1 := make(chan string, 1)
	var cs []chan string
	upgrader := newTestWebsocketUpgrader(func(ctx echo.Context, conn *testWebSocketConn) {
		cx := make(chan string, 1)
		cs = append(cs, cx)
		t1 <- "NEW"
		go func() {
			assert.Equal(t, "REQUEST", <-cx)
			err := conn.clientWrite([]byte("{ \"height\": abcd }"))
			assert.NoError(t, err)

			bs, err := conn.clientRead()
			assert.NoError(t, err)

			var res WSResponse
			err = json.Unmarshal(bs, &res)
			assert.NoError(t, err)
			t1 <- fmt.Sprint("RESULT:", res.Code)
		}()
	})
	s1 := make(chan string, 1)
	wm := newWSSessionManagerWithUpgrader(logger, 2, upgrader)
	chain := newTestChain(0,
		func(h int64) (getBlockFunc, error) {
			t.Logf("GetBlock(%d)", h)
			return func() module.Block {
				_, _ = <-s1
				return &testBlock{
					height: h,
					result: "empty",
				}
			}, nil
		},
		blockReceipts{
			"empty": testReceiptList{},
		},
	)
	go wm.RunEventSession(newTestContext(chain))
	assert.Equal(t, "NEW", <-t1)
	cs[0] <- "REQUEST"

	assert.Equal(t, fmt.Sprint("RESULT:", int(jsonrpc.ErrorCodeJsonParse)), <-t1)
	close(s1)
}

func TestWSSessionManager_MaxSession(t *testing.T) {
	logger := log.New()
	logger.SetOutput(io.Discard)

	t1 := make(chan string, 1)
	var cs []chan string

	upgrader := newTestWebsocketUpgrader(func(ctx echo.Context, conn *testWebSocketConn) {
		cx := make(chan string, 1)
		cs = append(cs, cx)
		t1 <- "NEW"
		t.Logf("[%p] NEW", conn)
		go func() {
			assert.Equal(t, "REQUEST", <-cx)
			t.Logf("[%p] REQUEST", conn)
			err := conn.clientWriteJSON(map[string]interface{}{
				"height": "0x1",
				"event":  "EventLog()",
				"logs":   "0x1",
			})
			assert.NoError(t, err)

			bs, err := conn.clientRead()
			assert.NoError(t, err)

			var res WSResponse
			err = json.Unmarshal(bs, &res)
			assert.NoError(t, err)
			t1 <- fmt.Sprint("RESULT:", res.Code)

			go func() {
				for value := <-cx; value == "WAIT"; value = <-cx {
					t.Logf("[%p] WAIT notification or result", conn)
					if bs, err := conn.clientRead(); err != nil {
						t.Logf("[%p] STOP msg=%+v", conn, err)
						break
					} else {
						t.Logf("[%p] RECEIVED msg=%s\n", conn, bs)
						t1 <- string(bs)
					}
				}
				t.Logf("[%p] CLOSE", conn)
				conn.Close()
				t1 <- "CLOSED"
			}()
		}()
	})
	s1 := make(chan string, 1)
	wm := newWSSessionManagerWithUpgrader(logger, 3, upgrader)
	chain1 := newTestChain(0,
		func(h int64) (getBlockFunc, error) {
			t.Logf("GetBlock(%d)", h)
			return func() module.Block {
				_, _ = <-s1
				return &testBlock{
					height: h,
					result: "empty",
				}
			}, nil
		},
		blockReceipts{
			"empty": testReceiptList{},
		},
	)
	chain2 := newTestChain(0,
		func(h int64) (getBlockFunc, error) {
			t.Logf("GetBlock(%d)", h)
			return func() module.Block {
				_, _ = <-s1
				return &testBlock{
					height: h,
					result: "empty",
				}
			}, nil
		},
		blockReceipts{
			"empty": testReceiptList{},
		},
	)
	go wm.RunEventSession(newTestContext(chain1))
	assert.Equal(t, "NEW", <-t1)
	go wm.RunEventSession(newTestContext(chain2))
	assert.Equal(t, "NEW", <-t1)
	go wm.RunEventSession(newTestContext(chain1))
	assert.Equal(t, "NEW", <-t1)
	go wm.RunEventSession(newTestContext(chain1))
	assert.Equal(t, "NEW", <-t1)

	cs[0] <- "REQUEST"
	assert.Equal(t, "RESULT:0", <-t1)

	cs[1] <- "REQUEST"
	assert.Equal(t, "RESULT:0", <-t1)

	cs[2] <- "REQUEST"
	assert.Equal(t, "RESULT:0", <-t1)

	cs[3] <- "REQUEST"
	assert.Equal(t, fmt.Sprint("RESULT:", int(jsonrpc.ErrorLackOfResource)), <-t1)

	cs[3] <- "QUIT"
	assert.Equal(t, "CLOSED", <-t1)

	cs[0] <- "WAIT"
	cs[1] <- "WAIT"
	cs[2] <- "WAIT"

	wm.StopSessionsForChain(chain2)
	assert.Equal(t, "CLOSED", <-t1)

	to := time.After(time.Millisecond * 100)
	select {
	case <-to:
	case msg := <-t1:
		t.Logf("MSG=%s", msg)
		t.Fail()
	}

	// let them be closed
	wm.SetMaxSession(0)
	assert.Equal(t, "CLOSED", <-t1)
	assert.Equal(t, "CLOSED", <-t1)
	close(s1)

	wm.StopAllSessions()
}

func TestWSSessionManager_Basic(t *testing.T) {
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
				"event":  "EventLog()",
				"logs":   "0x1",
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
						assert.NoError(t, err)
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
				newTestEventLog("cx01", "EventLog()", nil, nil),
			}),
		},
	}
	chain := newTestChain(0,
		func(h int64) (getBlockFunc, error) {
			t.Logf("GetBlock(%d)", h)
			if h < 10 {
				return func() module.Block {
					return &testBlock{
						height: h,
						result: "empty",
					}
				}, nil
			} else {
				return func() module.Block {
					_, _ = <-s1
					return &testBlock{
						height: h,
						result: "1",
						lb:     blkReceipts["1"].LogsBloom(),
					}
				}, nil
			}
		},
		blkReceipts,
	)
	go wm.RunEventSession(newTestContext(chain))

	// wait for the client and send request
	assert.Equal(t, "NEW", <-t1)
	c1 <- "REQUEST"

	// wait for the result
	assert.Equal(t, "RESULT:0", <-t1)

	// wait a message (1)
	c1 <- "WAIT"
	s1 <- "OK"
	t.Log(<-t1)

	// wait a message (2)
	s1 <- "OK"
	c1 <- "WAIT"
	t.Log(<-t1)

	// client quit
	c1 <- "QUIT"

	// wait for CLOSED
	assert.Equal(t, "CLOSED", <-t1)
	close(s1)

	wm.StopAllSessions()
}
