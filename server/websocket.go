package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/txresult"
	"github.com/labstack/echo/v4"
)

type wsSession struct {
	c     *websocket.Conn
	chain module.Chain
}

type wsSessionManager struct {
	sync.Mutex
	maxSession int
	sessions   []*wsSession
}

func newWSSessionManager() *wsSessionManager {
	return &wsSessionManager{
		maxSession: configMaxSession,
	}
}

func (wm *wsSessionManager) NewSession(c *websocket.Conn, chain module.Chain) *wsSession {
	wm.Lock()
	defer wm.Unlock()

	if len(wm.sessions) >= wm.maxSession {
		return nil
	}
	wss := &wsSession{c, chain}
	wm.sessions = append(wm.sessions, wss)
	return wss
}

func (wm *wsSessionManager) stopSessionAt(i int) {
	wss := wm.sessions[i]
	if wss.c != nil {
		wss.c.Close()
		wss.c = nil
	}
	last := len(wm.sessions) - 1
	wm.sessions[i] = wm.sessions[last]
	wm.sessions[last] = nil
	wm.sessions = wm.sessions[:last]
}

func (wm *wsSessionManager) StopSession(wss *wsSession) {
	wm.Lock()
	defer wm.Unlock()

	for i := 0; i < len(wm.sessions); i++ {
		if wss == wm.sessions[i] {
			wm.stopSessionAt(i)
		}
	}
}

func (wm *wsSessionManager) StopAllSessions() {
	wm.Lock()
	defer wm.Unlock()

	for i := 0; i < len(wm.sessions); i++ {
		wss := wm.sessions[i]
		if wss.c != nil {
			wss.c.Close()
			wss.c = nil
		}
	}
	wm.sessions = nil
}

func (wm *wsSessionManager) StopSessionsForChain(chain module.Chain) {
	wm.Lock()
	defer wm.Unlock()

	for i := 0; i < len(wm.sessions); i++ {
		wss := wm.sessions[i]
		if wss.chain == chain {
			wm.stopSessionAt(i)
		}
	}
}

func (wm *wsSessionManager) RunBlockSession(ctx echo.Context) error {
	chain, ok := ctx.Get("chain").(module.Chain)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "bad chain name")
	}

	upgrader := Upgrader()
	c, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return err
	}

	var wsResponse WSResponse
	wss := wm.NewSession(c, chain)
	if wss == nil {
		wsResponse.Code = int(jsonrpc.ErrorLackOfResource)
		wsResponse.Message = "too many monitor"
		c.WriteJSON(&wsResponse)
		c.Close()
		return nil
	}
	defer func() {
		wm.StopSession(wss)
	}()

	_, msgBS, err := c.ReadMessage()
	if err != nil {
		ctx.Logger().Error(err)
		return nil
	}
	var blockRequest BlockRequest
	if err := json.Unmarshal(msgBS, &blockRequest); err != nil {
		wsResponse.Code = int(jsonrpc.ErrorCodeJsonParse)
		wsResponse.Message = "bad block request"
		c.WriteJSON(&wsResponse)
		return nil
	}

	wsResponse.Code = 0
	c.WriteJSON(&wsResponse)

	ech := make(chan error)
	go readLoop(c, ech)

	h := blockRequest.Height.Value
	var bch <-chan module.Block
loop:
	for {
		bch, err = chain.BlockManager().WaitForBlock(h)
		if err != nil {
			wsResponse.Code = int(jsonrpc.ErrorCodeSystem)
			wsResponse.Message = err.Error()
			c.WriteJSON(&wsResponse)
			break loop
		}
		select {
		case err = <-ech:
			break loop
		case blk := <-bch:
			var blockNotification BlockNotification
			blockNotification.Height.Value = h
			blockNotification.Hash = blk.ID()
			err := c.WriteJSON(&blockNotification)
			if err != nil {
				break loop
			}
		}
		h++
	}
	ctx.Logger().Error(err)
	return nil
}

func (wm *wsSessionManager) RunEventSession(ctx echo.Context) error {
	chain, ok := ctx.Get("chain").(module.Chain)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "bad chain name")
	}

	upgrader := Upgrader()
	c, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return err
	}

	var wsResponse WSResponse
	wss := wm.NewSession(c, chain)
	if wss == nil {
		wsResponse.Code = int(jsonrpc.ErrorLackOfResource)
		wsResponse.Message = "too many monitor"
		c.WriteJSON(&wsResponse)
		c.Close()
	}
	defer func() {
		wm.StopSession(wss)
	}()

	_, msgBS, err := c.ReadMessage()
	if err != nil {
		ctx.Logger().Error(err)
		return nil
	}
	var er EventRequest
	if err := json.Unmarshal(msgBS, &er); err != nil {
		wsResponse.Code = int(jsonrpc.ErrorCodeJsonParse)
		wsResponse.Message = "bad event request"
		c.WriteJSON(&wsResponse)
		return nil
	}
	lb, err := er.compile()
	if err != nil {
		wsResponse.Code = int(jsonrpc.ErrorCodeInvalidParams)
		wsResponse.Message = "bad event request parameter"
		c.WriteJSON(&wsResponse)
		return nil
	}

	wsResponse.Code = 0
	c.WriteJSON(&wsResponse)

	ech := make(chan error)
	go readLoop(c, ech)

	h := er.Height.Value
	var bch <-chan module.Block
loop:
	for {
		bch, err = chain.BlockManager().WaitForBlock(h)
		if err != nil {
			wsResponse.Code = int(jsonrpc.ErrorCodeSystem)
			wsResponse.Message = err.Error()
			c.WriteJSON(&wsResponse)
			break loop
		}
		select {
		case err = <-ech:
			break loop
		case blk := <-bch:
			if !blk.LogsBloom().Contain(lb) {
				h++
				continue loop
			}
			rl, err := chain.ServiceManager().ReceiptListFromResult(blk.Result(), module.TransactionGroupNormal)
			if err != nil {
				wsResponse.Code = int(jsonrpc.ErrorCodeSystem)
				wsResponse.Message = err.Error()
				c.WriteJSON(&wsResponse)
				break loop
			}
			index := int32(0)
			for rit := rl.Iterator(); rit.Has(); rit.Next() {
				r, err := rit.Get()
				if err != nil {
					wsResponse.Code = int(jsonrpc.ErrorCodeSystem)
					wsResponse.Message = err.Error()
					c.WriteJSON(&wsResponse)
					break loop
				}
				if r.LogsBloom().Contain(lb) {
					for eit := r.EventLogIterator(); eit.Has(); eit.Next() {
						e, err := eit.Get()
						if err != nil {
							wsResponse.Code = int(jsonrpc.ErrorCodeSystem)
							wsResponse.Message = err.Error()
							c.WriteJSON(&wsResponse)
							break loop
						}
						if er.match(e) {
							var eventNotification EventNotification
							eventNotification.Height.Value = h
							eventNotification.Hash = blk.ID()
							eventNotification.Index.Value = index
							err := c.WriteJSON(&eventNotification)
							if err != nil {
								break loop
							}
							break
						}
					}
				}
				index++
			}
		}
		h++
	}
	ctx.Logger().Error(err)
	return nil
}

const configMaxSession = 10

type BlockRequest struct {
	Height common.HexInt64 `json:"height"`
}

type EventRequest struct {
	Height  common.HexInt64 `json:"height"`
	Addr    *common.Address `json:"addr"`
	Event   string          `json:"event"`
	Data    []interface{}   `json:"data"`
	dataBSs [][]byte
}

type WSResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

type BlockNotification struct {
	Hash   common.HexBytes `json:"hash"`
	Height common.HexInt64 `json:"height"`
}

type EventNotification struct {
	Hash   common.HexBytes `json:"hash"`
	Height common.HexInt64 `json:"height"`
	Index  common.HexInt32 `json:"index"`
}

func Upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{}
}

func (er *EventRequest) compile() (module.LogsBloom, error) {
	lb := txresult.NewLogsBloom(nil)
	if er.Addr != nil {
		lb.AddAddressOfLog(er.Addr)
	}
	name, typeStr := txresult.DecomposeEventSignature(er.Event)
	if len(name) == 0 || typeStr == nil || len(typeStr) < len(er.Data) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "bad event request")
	}
	lb.AddIndexedOfLog(0, []byte(er.Event))
	er.dataBSs = make([][]byte, len(er.Data))
	for i, d := range er.Data {
		if d != nil {
			dStr := d.(string)
			bs, err := txresult.EventDataStringToBytesByType(typeStr[i], dStr)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusBadRequest, "bad event data")
			}
			lb.AddIndexedOfLog(i+1, bs)
			er.dataBSs[i] = bs
		}
	}
	return lb, nil
}

func (er *EventRequest) match(el module.EventLog) bool {
	if !bytes.Equal([]byte(er.Event), el.Indexed()[0]) {
		return false
	}
	if er.Addr != nil && !el.Address().Equal(er.Addr) {
		return false
	}
	for i, d := range er.Data {
		if d != nil {
			if len(el.Indexed()) <= i+1 {
				return false
			}
			if !bytes.Equal(er.dataBSs[i], el.Indexed()[i+1]) {
				return false
			}
		}
	}
	return true
}

func readLoop(c *websocket.Conn, ech chan<- error) {
	for {
		if _, _, err := c.NextReader(); err != nil {
			ech <- err
			break
		}
	}
}
