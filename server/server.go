package server

import (
	"context"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/server/v3"
)

type Manager struct {
	e     *echo.Echo
	chain *module.Chain // Chain Manager
}

func NewManager() *Manager {

	e := echo.New()

	validator := jsonrpc.NewValidator()
	v3.RegisterValidationRule(validator)

	e.HideBanner = true
	e.HTTPErrorHandler = jsonrpc.ErrorHandler
	e.Validator = validator

	return &Manager{
		e: e,
	}
}

func (srv *Manager) SetChain(chain module.Chain) {
	srv.chain = &chain
}

func (srv *Manager) Chain() *module.Chain {
	return srv.chain
}

func (srv *Manager) Start() {
	e := srv.e

	// method
	mr := v3.MethodRepository()

	// middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// jsonrpc
	g := e.Group("/api")
	g.Use(JsonRpc(srv, mr), JsonRpcLogger(), Chunk())
	g.POST("/v3", mr.Handle)
	g.POST("/v3/:channel", mr.Handle)

	// websocket
	e.GET("/ws/echo", wsEcho)

	// metric
	// e.GET("/metrics", echo.WrapHandler(metric.PromethusExporter()))

	// Start server
	go func() {
		// cfg.rpc_port
		if err := e.Start(":9081"); err != nil {
			e.Logger.Info("shutting down the server")
		}
	}()
}

func (srv *Manager) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := srv.e.Shutdown(ctx); err != nil {
		srv.e.Logger.Fatal(err)
	}
}
