package server

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/txresult"
)

var nSessions int32

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

func (er *EventRequest) compile() (module.LogBloom, error) {
	lb := txresult.NewLogBloom(nil)
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

func wsEcho(c echo.Context) error {

	upgrader := Upgrader()
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	for {
		// Write
		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
		if err != nil {
			c.Logger().Error(err)
		}

		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}
		log.Printf("%s\n", msg)
	}
}

func readLoop(c *websocket.Conn, ech chan<- error) {
	for {
		if _, _, err := c.NextReader(); err != nil {
			ech <- err
			break
		}
	}
}

func wsBlock(ctx echo.Context) error {
	chain, ok := ctx.Get("chain").(module.Chain)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "bad chain name")
	}

	var started bool
	if atomic.AddInt32(&nSessions, 1) > configMaxSession {
		atomic.AddInt32(&nSessions, -1)
		return echo.NewHTTPError(http.StatusTooManyRequests, "too many stream sessions")
	}
	defer func() {
		if !started {
			atomic.AddInt32(&nSessions, -1)
		}
	}()

	upgrader := Upgrader()
	c, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if !started {
			c.Close()
		}
	}()

	_, msgBS, err := c.ReadMessage()
	if err != nil {
		return err
	}
	var blockRequest BlockRequest
	if err := json.Unmarshal(msgBS, &blockRequest); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bad block request")
	}

	ech := make(chan error)
	go readLoop(c, ech)
	go func() {
		h := blockRequest.Height.Value
		var err error
		var bch <-chan module.Block
	loop:
		for {
			bch, err = chain.BlockManager().WaitForBlock(h)
			if err != nil {
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
		atomic.AddInt32(&nSessions, -1)
		c.Close()
	}()

	started = true
	return nil
}

func wsEvent(ctx echo.Context) error {
	chain, ok := ctx.Get("chain").(module.Chain)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "bad chain name")
	}

	var started bool
	if atomic.AddInt32(&nSessions, 1) > configMaxSession {
		atomic.AddInt32(&nSessions, -1)
		return echo.NewHTTPError(http.StatusTooManyRequests, "too many stream sessions")
	}
	defer func() {
		if !started {
			atomic.AddInt32(&nSessions, -1)
		}
	}()

	upgrader := Upgrader()
	c, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if !started {
			c.Close()
		}
	}()

	_, msgBS, err := c.ReadMessage()
	if err != nil {
		return err
	}
	var er EventRequest
	if err := json.Unmarshal(msgBS, &er); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bad event request")
	}
	lb, err := er.compile()

	ech := make(chan error)
	go readLoop(c, ech)
	go func() {
		h := er.Height.Value
		var err error
		var bch <-chan module.Block
	loop:
		for {
			bch, err = chain.BlockManager().WaitForBlock(h)
			if err != nil {
				break loop
			}
			select {
			case err = <-ech:
				break loop
			case blk := <-bch:
				if !blk.LogBloom().Contain(lb) {
					h++
					continue loop
				}
				rl := chain.ServiceManager().ReceiptListFromResult(blk.Result(), module.TransactionGroupNormal)
				index := int32(0)
				for rit := rl.Iterator(); rit.Has(); rit.Next() {
					r, _ := rit.Get()
					if r.LogBloom().Contain(lb) {
						for eit := r.EventLogIterator(); eit.Has(); eit.Next() {
							e, _ := eit.Get()
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
		atomic.AddInt32(&nSessions, -1)
		c.Close()
	}()

	started = true
	return nil
}
