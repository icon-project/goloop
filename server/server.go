package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/server/metric"
	v3 "github.com/icon-project/goloop/server/v3"
)

const (
	flagENABLE  int32 = 1
	flagDISABLE int32 = 0
	UrlAdmin          = "/admin"
)

type Manager struct {
	e                     *echo.Echo
	addr                  string
	wallet                module.Wallet
	chains                map[string]module.Chain // chain manager
	wssm                  *wsSessionManager
	mtx                   sync.RWMutex
	jsonrpcDefaultChannel string
	jsonrpcMessageDump    int32
	jsonrpcIncludeDebug   int32
	logger                log.Logger
	metricsHandler        echo.HandlerFunc
}

func NewManager(addr string,
	jsonrpcDump bool,
	jsonrpcIncludeDebug bool,
	jsonrpcDefaultChannel string,
	wallet module.Wallet,
	l log.Logger) *Manager {

	e := echo.New()

	validator := jsonrpc.NewValidator()
	v3.RegisterValidationRule(validator)

	e.HideBanner = true
	e.HidePort = true

	e.HTTPErrorHandler = HTTPErrorHandler
	e.Validator = validator
	logger := l.WithFields(log.Fields{log.FieldKeyModule: "SR"})

	m := &Manager{
		e:                     e,
		addr:                  addr,
		wallet:                wallet,
		chains:                make(map[string]module.Chain),
		wssm:                  newWSSessionManager(logger),
		mtx:                   sync.RWMutex{},
		jsonrpcDefaultChannel: jsonrpcDefaultChannel,
		logger:                logger,
		metricsHandler:        echo.WrapHandler(metric.PrometheusExporter()),
	}
	m.SetMessageDump(jsonrpcDump)
	m.SetIncludeDebug(jsonrpcIncludeDebug)
	return m
}

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

func atomicStore(addr *int32, enable bool) {
	v := flagDISABLE
	if enable {
		v = flagENABLE
	}
	atomic.StoreInt32(addr, v)
}

func atomicLoad(addr *int32) bool {
	if atomic.LoadInt32(addr) == flagENABLE {
		return true
	}
	return false
}

func (srv *Manager) SetMessageDump(enable bool) {
	atomicStore(&srv.jsonrpcMessageDump, enable)
}

func (srv *Manager) MessageDump() bool {
	return atomicLoad(&srv.jsonrpcMessageDump)
}

func (srv *Manager) SetIncludeDebug(enable bool) {
	atomicStore(&srv.jsonrpcIncludeDebug, enable)
}

func (srv *Manager) IncludeDebug() bool {
	return atomicLoad(&srv.jsonrpcIncludeDebug)
}

func (srv *Manager) Start() error {
	srv.logger.Infoln("starting the server")
	// CORS middleware
	srv.e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		MaxAge: 3600,
	}))

	// json rpc
	srv.RegisterAPIHandler(srv.e.Group("/api"))

	// metric
	srv.RegisterMetricsHandler(srv.e.Group("/metrics"))

	return srv.e.Start(srv.addr)
}

func (srv *Manager) RegisterAPIHandler(g *echo.Group) {
	g.Use(middleware.Recover())

	// group for json rpc
	rpc := g.Group("")
	rpc.Use(middleware.BodyDump(func(c echo.Context, reqBody []byte, resBody []byte) {
		if srv.MessageDump() {
			srv.logger.Printf("request=%s", reqBody)
			srv.logger.Printf("response=%s", resBody)
		}
	}))
	rpc.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			ctx.Set("includeDebug", srv.IncludeDebug())
			return next(ctx)
		}
	})
	// group for websocket
	ws := g.Group("")

	// Config
	cfg := &jsonrpc.Config{}
	cfgFile := path.Join(cfg.AbsBaseDir(), "config.json")
	if st, err := os.Stat(cfgFile); err != nil || !st.Mode().IsRegular() {
		srv.logger.Printf("Error loading config: %s", err.Error())
	}
	b, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		srv.logger.Printf("Error loading config: %s", err.Error())
	}
	if err = json.Unmarshal(b, cfg); err != nil {
		srv.logger.Printf("Error loading config: %s", err.Error())
	}

	// v3 APIs
	mr := v3.MethodRepository(cfg)
	dmr := v3.DebugMethodRepository(cfg)
	v3api := rpc.Group("/v3")
	v3api.Use(JsonRpc(), Chunk())
	v3api.POST("", mr.Handle, ChainInjector(srv))
	v3api.POST("/", mr.Handle, ChainInjector(srv))
	v3api.POST("/:channel", mr.Handle, ChainInjector(srv))

	v3dbg := rpc.Group("/v3d")
	v3dbg.Use(srv.CheckDebug(), JsonRpc(), Chunk())
	v3dbg.POST("", dmr.Handle, ChainInjector(srv))
	v3dbg.POST("/", dmr.Handle, ChainInjector(srv))
	v3dbg.POST("/:channel", dmr.Handle, ChainInjector(srv))

	ws.GET("/v3/:channel/block", srv.wssm.RunBlockSession, ChainInjector(srv))
	ws.GET("/v3/:channel/event", srv.wssm.RunEventSession, ChainInjector(srv))
}

func (srv *Manager) RegisterMetricsHandler(g *echo.Group) {
	g.GET("", srv.metricsHandler)
}

func (srv *Manager) CheckDebug() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if !srv.IncludeDebug() {
				return ctx.String(http.StatusNotFound, "rpc_debug is false")
			}
			return next(ctx)
		}
	}
}

func (srv *Manager) Stop() error {
	srv.logger.Infoln("shutting down the server")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	srv.wssm.StopAllSessions()
	return srv.e.Shutdown(ctx)
}

func (srv *Manager) AdminEchoGroup(m ...echo.MiddlewareFunc) *echo.Group {
	return srv.e.Group(UrlAdmin, m...)
}
