package server

import (
	"context"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/server/v3"
)

type Manager struct {
	e      *echo.Echo
	addr   string
	chains map[string]*module.Chain // chain manager
	mtx    sync.RWMutex
}

func NewManager(addr string) *Manager {

	e := echo.New()

	validator := jsonrpc.NewValidator()
	v3.RegisterValidationRule(validator)

	e.HideBanner = true
	e.HTTPErrorHandler = jsonrpc.ErrorHandler
	e.Validator = validator

	return &Manager{
		e:      e,
		addr:   addr,
		chains: map[string]*module.Chain{},
		mtx:    sync.RWMutex{},
	}
}

// TODO : channel-chain
func (srv *Manager) SetChain(channel string, chain module.Chain) {
	if channel == "" || chain == nil {
		return
	}
	srv.mtx.Lock()
	srv.chains[channel] = &chain
	srv.mtx.Unlock()
}

func (srv *Manager) Chain(channel string) *module.Chain {
	if channel == "" {
		channel = "default"
	}
	srv.mtx.RLock()
	chain, ok := srv.chains[channel]
	if !ok {
		return nil
	}
	srv.mtx.RUnlock()
	return chain
}

func (srv *Manager) Start() {

	// middleware
	// srv.e.Use(middleware.Logger())
	srv.e.Use(middleware.Recover())

	// method
	mr := v3.MethodRepository()

	// jsonrpc
	g := srv.e.Group("/api")
	// g.Use(JsonRpc(srv, mr), JsonRpcLogger(), Chunk())
	g.Use(JsonRpc(srv, mr), Chunk())
	g.POST("/v3", mr.Handle)
	g.POST("/v3/:channel", mr.Handle)

	// websocket
	srv.e.GET("/ws/echo", wsEcho)

	// metric
	srv.e.GET("/metrics", echo.WrapHandler(metric.PromethusExporter()))

	// Start server : main loop
	if err := srv.e.Start(srv.addr); err != nil {
		srv.e.Logger.Info("shutting down the server")
	}
}

func (srv *Manager) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := srv.e.Shutdown(ctx); err != nil {
		srv.e.Logger.Fatal(err)
	}
}
