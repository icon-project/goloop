package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
)

type wsSession struct {
	c     *websocket.Conn
	chain module.Chain
}

type wsSessionManager struct {
	sync.Mutex
	maxSession int
	logger     log.Logger
	sessions   []*wsSession
}

func newWSSessionManager(logger log.Logger, maxSession int) *wsSessionManager {
	return &wsSessionManager{
		maxSession: maxSession,
		logger:     logger,
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

	wm.stopAllSessionsInLock()
}

func (wm *wsSessionManager) stopAllSessionsInLock() {
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

func (wm *wsSessionManager) SetMaxSession(limit int) {
	wm.Lock()
	defer wm.Unlock()

	wm.maxSession = limit
	if limit <= 0 {
		wm.stopAllSessionsInLock()
	}
}

func (wm *wsSessionManager) initSession(ctx echo.Context, reqPtr interface{}) (*wsSession, error) {
	u := Upgrader()
	c, err := u.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return nil, err
	}

	chain, err := wm.chain(ctx)
	if err != nil {
		return nil, err
	}

	_, msgBS, err := c.ReadMessage()
	if err != nil {
		wm.logger.Warnf("%+v\n", err)
		return nil, err
	}
	if err := json.Unmarshal(msgBS, reqPtr); err != nil {
		wsResponse := WSResponse{
			Code:    int(jsonrpc.ErrorCodeJsonParse),
			Message: "bad event request",
		}
		c.WriteJSON(&wsResponse)
		c.Close()
		return nil, err
	}

	wss := wm.NewSession(c, chain)
	if wss == nil {
		wsResponse := WSResponse{
			Code:    int(jsonrpc.ErrorLackOfResource),
			Message: "too many monitor",
		}
		c.WriteJSON(&wsResponse)
		c.Close()
		return nil, errors.New("too many monitor")
	}
	return wss, nil
}

func (wm *wsSessionManager) chain(ctx echo.Context) (module.Chain, error) {
	c, ok := ctx.Get("chain").(module.Chain)
	if !ok {
		return nil, errors.New("chain is not contained in this context")
	}
	return c, nil
}

func (wss *wsSession) response(code int, msg string) error {
	wsResponse := WSResponse{
		Code:    code,
		Message: msg,
	}
	return wss.WriteJSON(&wsResponse)
}

func (wss *wsSession) WriteJSON(v interface{}) error {
	return wss.c.WriteJSON(v)
}

const DefaultWSMaxSession = 10

type WSResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

func Upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
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
