package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/server/v3"
)

type Manager struct {
	e                     *echo.Echo
	addr                  string
	wallet                module.Wallet
	chains                map[string]module.Chain // chain manager
	wssm                  *wsSessionManager
	mtx                   sync.RWMutex
	jsonrpcDump           bool
	jsonrpcDefaultChannel string
	logger                log.Logger
}

func NewManager(addr string, jsonrpcDump bool, jsonrpcDefaultChannel string, wallet module.Wallet, l log.Logger) *Manager {

	e := echo.New()

	validator := jsonrpc.NewValidator()
	v3.RegisterValidationRule(validator)

	e.HideBanner = true
	e.HidePort = true

	e.HTTPErrorHandler = HTTPErrorHandler
	e.Validator = validator
	logger := l.WithFields(log.Fields{log.FieldKeyModule: "SR"})

	return &Manager{
		e:           e,
		addr:        addr,
		wallet:      wallet,
		chains:      make(map[string]module.Chain),
		wssm:        newWSSessionManager(logger),
		mtx:         sync.RWMutex{},
		jsonrpcDump: jsonrpcDump,
		jsonrpcDefaultChannel: jsonrpcDefaultChannel,
		logger:      logger,
	}
}

// TODO : channel-chain
func (srv *Manager) SetChain(channel string, chain module.Chain) {
	defer srv.mtx.Unlock()
	srv.mtx.Lock()

	if channel == "" || chain == nil {
		return
	}
	srv.chains[channel] = chain
}

func (srv *Manager) RemoveChain(channel string) {
	defer srv.mtx.Unlock()
	srv.mtx.Lock()

	if channel == "" {
		return
	}
	if chain, ok := srv.chains[channel]; ok {
		srv.wssm.StopSessionsForChain(chain)
		delete(srv.chains, channel)
	}
}

func (srv *Manager) Chain(channel string) module.Chain {
	defer srv.mtx.RUnlock()
	srv.mtx.RLock()

	if channel == "" {
		if srv.jsonrpcDefaultChannel == "" && len(srv.chains) == 1 {
			for k := range srv.chains {
				channel = k
			}
		} else {
			channel = srv.jsonrpcDefaultChannel
		}
	}
	return srv.chains[channel]
}

func (srv *Manager) SetDefaultChannel(jsonrpcDefaultChannel string) {
	defer srv.mtx.Unlock()
	srv.mtx.Lock()

	srv.jsonrpcDefaultChannel = jsonrpcDefaultChannel
}

func (srv *Manager) Start() {
	srv.logger.Infoln("starting the server")
	// middleware
	// srv.e.Use(middleware.Logger())
	srv.e.Use(middleware.Recover())
	srv.e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		MaxAge: 3600,
	}))

	// auth : hello test
	srv.e.POST("/auth", authentication(srv.wallet))
	srv.e.GET("hello", func(c echo.Context) error {
		token := c.Get("token").(*jwt.Token)
		claims := token.Claims.(*tokenClaims)
		return c.JSON(http.StatusOK, fmt.Sprintf("Hello: %s[%s]", claims.Audience, claims.Role))
	}, middleware.JWTWithConfig(JWTConfig(srv.wallet)))

	// method
	mr := v3.MethodRepository()

	// jsonrpc
	g := srv.e.Group("/api")
	if srv.jsonrpcDump {
		g.Use(middleware.BodyDump(func(c echo.Context, reqBody []byte, resBody []byte) {
			srv.logger.Printf("request=%s", reqBody)
			srv.logger.Printf("respose=%s", resBody)
		}))
	}
	g.Use(JsonRpc(mr), Chunk())
	g.POST("/v3", mr.Handle, ChainInjector(srv))
	g.POST("/v3/:channel", mr.Handle, ChainInjector(srv))

	// websocket
	srv.e.GET("/api/v3/:channel/block", srv.wssm.RunBlockSession, ChainInjector(srv))
	srv.e.GET("/api/v3/:channel/event", srv.wssm.RunEventSession, ChainInjector(srv))

	// metric
	srv.e.GET("/metrics", echo.WrapHandler(metric.PromethusExporter()))

	// document: redoc
	// opts := RedocOpts{
	// 	SpecURL: "doc/swagger.yaml",
	// }
	// srv.e.GET("/doc", Redoc(opts))
	// srv.e.File("doc/swagger.yaml", "./doc/swagger.yaml")

	// Start server : main loop
	if err := srv.e.Start(srv.addr); err != nil {
		srv.logger.Infoln("shutting down the server")
	}
}

func (srv *Manager) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	srv.wssm.StopAllSessions()
	if err := srv.e.Shutdown(ctx); err != nil {
		srv.e.Logger.Fatal(err)
	}
}

func (srv *Manager) AdminEchoGroup() *echo.Group {
	return srv.e.Group("/admin")
}
