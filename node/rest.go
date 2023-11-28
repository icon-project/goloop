package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"syscall"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/chain"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/server"
	"github.com/icon-project/goloop/server/metric"
	"github.com/icon-project/goloop/service"
)

const (
	UrlSystem   = "/system"
	UrlUser     = "/user"
	UrlStats    = "/stats"
	UrlChain    = "/chain"
	ParamCID    = "cid"
	UrlChainRes = "/:" + ParamCID
	ParamID     = "id"
	UrlUserRes  = "/:" + ParamID
	TaskID      = "task"

	UrlDB    = "/db"
	ParamBK  = "bucket"
	ParamKey = "key"
)

type Rest struct {
	n *Node
	a *Auth
}

type SystemView struct {
	BuildVersion string `json:"buildVersion"`
	BuildTags    string `json:"buildTags"`
	Setting      struct {
		Address       string `json:"address"`
		P2PAddr       string `json:"p2p"`
		P2PListenAddr string `json:"p2pListen"`
		RPCAddr       string `json:"rpcAddr"`
		RPCDump       bool   `json:"rpcDump"`
	} `json:"setting"`
	Config interface{} `json:"config"`
}

type StatsView struct {
	Chains    []map[string]interface{} `json:"chains"`
	Timestamp time.Time                `json:"timestamp"`
}

type ChainView struct {
	CID       common.HexInt32 `json:"cid"`
	NID       common.HexInt32 `json:"nid"`
	Channel   string          `json:"channel"`
	State     string          `json:"state"`
	Height    int64           `json:"height"`
	LastError string          `json:"lastError"`
}

type ChainInspectView struct {
	*ChainView
	GenesisTx json.RawMessage `json:"genesisTx"`
	Config    *ChainConfig    `json:"config"`
	// TODO [TBD] define structure each module for inspect
	Module map[string]interface{} `json:"module"`
}

type ChainConfig struct {
	DBType           string `json:"dbType"`
	Platform         string `json:"platform"`
	SeedAddr         string `json:"seedAddress"`
	Role             uint   `json:"role"`
	ConcurrencyLevel int    `json:"concurrencyLevel,omitempty"`
	NormalTxPoolSize int    `json:"normalTxPool,omitempty"`
	PatchTxPoolSize  int    `json:"patchTxPool,omitempty"`
	MaxBlockTxBytes  int    `json:"maxBlockTxBytes,omitempty"`
	NodeCache        string `json:"nodeCache,omitempty"`
	Channel          string `json:"channel"`
	SecureSuites     string `json:"secureSuites"`
	SecureAeads      string `json:"secureAeads"`
	DefWaitTimeout   int64  `json:"defaultWaitTimeout"`
	MaxWaitTimeout   int64  `json:"maxWaitTimeout"`
	TxTimeout        int64  `json:"txTimeout"`
	AutoStart        bool   `json:"autoStart"`
	ChildrenLimit    *int   `json:"childrenLimit,omitempty"`
	NephewsLimit     *int   `json:"nephewsLimit,omitempty"`
	ValidateTxOnSend bool   `json:"validateTxOnSend,omitempty"`
}

type ChainResetParam struct {
	Height    int64           `json:"height,omitempty"`
	BlockHash common.HexBytes `json:"blockHash,omitempty"`
}

type ChainImportParam struct {
	DBPath string `json:"dbPath"`
	Height int64  `json:"height"`
}

type ChainPruneParam struct {
	DBType string `json:"dbType,omitempty"`
	Height int64  `json:"height"`
}

type ChainBackupParam struct {
	Manual bool `json:"manual,omitempty"`
}

type ConfigureParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RestoreBackupParam struct {
	Name      string `json:"name"`
	Overwrite bool   `json:"overwrite"`
}

func NewChainView(c *Chain) *ChainView {
	state, height, lastErr := c.State()
	v := &ChainView{
		CID:     common.HexInt32{Value: int32(c.CID())},
		NID:     common.HexInt32{Value: int32(c.NID())},
		Channel: c.Channel(),
		State:   state,
		Height:  height,
	}
	if lastErr != nil {
		v.LastError = lastErr.Error()
	}
	return v
}

type InspectFunc func(c module.Chain, informal bool) map[string]interface{}

var (
	inspectFuncs = make(map[string]InspectFunc)
)

func NewChainInspectView(c *Chain) *ChainInspectView {
	v := &ChainInspectView{
		ChainView: NewChainView(c),
		GenesisTx: c.Genesis(),
		Config:    NewChainConfig(c.cfg),
	}
	return v
}

func NewChainConfig(cfg *chain.Config) *ChainConfig {
	v := &ChainConfig{
		DBType:           cfg.DBType,
		Platform:         cfg.Platform,
		SeedAddr:         cfg.SeedAddr,
		Role:             cfg.Role,
		ConcurrencyLevel: cfg.ConcurrencyLevel,
		NormalTxPoolSize: cfg.NormalTxPoolSize,
		PatchTxPoolSize:  cfg.PatchTxPoolSize,
		MaxBlockTxBytes:  cfg.MaxBlockTxBytes,
		NodeCache:        cfg.NodeCache,
		Channel:          cfg.Channel,
		SecureSuites:     cfg.SecureSuites,
		SecureAeads:      cfg.SecureAeads,
		DefWaitTimeout:   cfg.DefWaitTimeout,
		MaxWaitTimeout:   cfg.MaxWaitTimeout,
		TxTimeout:        cfg.TxTimeout,
		AutoStart:        cfg.AutoStart,
		ChildrenLimit:    cfg.ChildrenLimit,
		NephewsLimit:     cfg.NephewsLimit,
		ValidateTxOnSend: cfg.ValidateTxOnSend,
	}
	return v
}

func RegisterInspectFunc(name string, f InspectFunc) error {
	if _, ok := inspectFuncs[name]; ok {
		return fmt.Errorf("already exist function name:%s", name)
	}
	inspectFuncs[name] = f
	return nil
}

func RegisterRest(n *Node) {
	r := Rest{
		n: n,
		a: NewAuth(path.Join(n.cfg.ResolveAbsolute(n.cfg.BaseDir), "auth.json"), server.UrlAdmin),
	}
	r.a.SkipIfEmptyUsers = n.cfg.AuthSkipIfEmptyUsers
	ag := n.srv.AdminEchoGroup(r.a.MiddlewareFunc())
	r.RegisterChainHandlers(ag.Group(UrlChain))
	r.RegisterSystemHandlers(ag.Group(UrlSystem))

	r.RegisterChainHandlers(n.cliSrv.e.Group(UrlChain))
	r.RegisterSystemHandlers(n.cliSrv.e.Group(UrlSystem))
	r.RegisterUserHandlers(n.cliSrv.e.Group(UrlUser))
	r.RegisterStatsHandlers(n.cliSrv.e.Group(UrlStats))
	r.RegisterDBHandlers(n.cliSrv.e.Group(UrlDB))

	_ = RegisterInspectFunc("metrics", metric.Inspect)
	_ = RegisterInspectFunc("network", network.Inspect)
	_ = RegisterInspectFunc("service", service.Inspect)

	// json rpc
	n.srv.RegisterAPIHandler(n.cliSrv.e.Group("/api"))

	// metric
	n.srv.RegisterMetricsHandler(n.cliSrv.e.Group("/metrics"))
}

func (r *Rest) RegisterChainHandlers(g *echo.Group) {
	g.GET("", r.GetChains)
	g.POST("", r.JoinChain)

	g.GET(UrlChainRes, r.GetChain, r.ChainInjector)
	g.DELETE(UrlChainRes, r.LeaveChain, r.ChainInjector)
	g.POST(UrlChainRes+"/start", r.StartChain, r.ChainInjector)
	g.POST(UrlChainRes+"/stop", r.StopChain, r.ChainInjector)
	g.POST(UrlChainRes+"/reset", r.ResetChain, r.ChainInjector)
	g.POST(UrlChainRes+"/verify", r.VerifyChain, r.ChainInjector)
	g.POST(UrlChainRes+"/import", r.ImportChain, r.ChainInjector)
	g.POST(UrlChainRes+"/prune", r.PruneChain, r.ChainInjector)
	g.POST(UrlChainRes+"/backup", r.BackupChain, r.ChainInjector)
	route := g.GET(UrlChainRes+"/genesis", r.GetChainGenesis, r.ChainInjector)
	if r.a != nil {
		r.a.SetSkip(route, false)
	}
	g.GET(UrlChainRes+"/configure", r.GetChainConfig, r.ChainInjector)
	g.POST(UrlChainRes+"/configure", r.ConfigureChain, r.ChainInjector)
	g.POST(UrlChainRes+"/:"+TaskID, r.RunChainTask, r.ChainInjector)
}

func (r *Rest) ChainInjector(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		p := ctx.Param(ParamCID)
		c := r.n.GetChainBySelector(p)
		if c == nil {
			return ctx.String(http.StatusNotFound,
				fmt.Sprintf("Chain(%s: cid or channel) not found", p))
		}
		ctx.Set("chain", c)
		return next(ctx)
	}
}

func (r *Rest) GetChains(ctx echo.Context) error {
	l := make([]*ChainView, 0)
	for _, c := range r.n.GetChains() {
		v := NewChainView(c)
		l = append(l, v)
	}
	return ctx.JSON(http.StatusOK, l)
}

func GetJsonMultipart(ctx echo.Context, ptr interface{}) error {
	jsonStr := ctx.FormValue("json")
	if err := json.Unmarshal([]byte(jsonStr), ptr); err != nil {
		return err
	}
	return nil
}

func GetFileMultipart(ctx echo.Context, fieldname string) ([]byte, error) {
	ff, err := ctx.FormFile(fieldname)
	if err != nil {
		return nil, err
	}
	f, err := ff.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Rest) JoinChain(ctx echo.Context) error {
	p := &ChainConfig{}

	if err := GetJsonMultipart(ctx, p); err != nil {
		return errors.Wrap(err, "fail to get 'json' from multipart")
	}

	genesis, err := GetFileMultipart(ctx, "genesisZip")
	if err != nil {
		return errors.Wrap(err, "fail to get 'genesisZip' from multipart")
	}

	c, err := r.n.JoinChain(p, genesis)
	if err != nil {
		if we, ok := err.(errors.Unwrapper); ok {
			switch we.Unwrap() {
			case ErrAlreadyExists:
				return ctx.String(http.StatusConflict, err.Error())
			}
		}
		return errors.Wrap(err, "fail to join")
	}
	return ctx.String(http.StatusOK, fmt.Sprintf("%#x", c.CID()))
}

var (
	defaultJsonTemplate = NewJsonTemplate("default")
)

func (r *Rest) GetChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	v := NewChainInspectView(c)

	informal, _ := strconv.ParseBool(ctx.QueryParam("informal"))
	v.Module = make(map[string]interface{})
	for name, f := range inspectFuncs {
		if m := f(c, informal); m != nil {
			v.Module[name] = m
		}
	}
	format := ctx.QueryParam("format")
	if format != "" {
		return defaultJsonTemplate.Response(format, v, ctx.Response())
	}
	return ctx.JSON(http.StatusOK, v)
}

func (r *Rest) LeaveChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.LeaveChain(c.CID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) StartChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.StartChain(c.CID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) StopChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.StopChain(c.CID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) ResetChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	param := &ChainResetParam{}
	if err := ctx.Bind(param); err != nil {
		return echo.ErrBadRequest
	}
	if param.Height < 0 {
		return echo.ErrBadRequest
	}
	if err := r.n.ResetChain(c.CID(), param.Height, param.BlockHash); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) VerifyChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	if err := r.n.VerifyChain(c.CID()); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) ImportChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	param := &ChainImportParam{}
	if err := ctx.Bind(param); err != nil {
		return echo.ErrBadRequest
	}
	if err := r.n.ImportChain(c.CID(), param.DBPath, param.Height); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) PruneChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	param := &ChainPruneParam{}
	if err := ctx.Bind(param); err != nil {
		return echo.ErrBadRequest
	}
	if param.Height < 1 {
		return echo.ErrBadRequest
	}
	if err := r.n.PruneChain(c.CID(), param.DBType, param.Height); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) BackupChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	param := &ChainBackupParam{}
	if err := ctx.Bind(param); err != nil {
		return echo.ErrBadRequest
	}
	if name, err := r.n.BackupChain(c.CID(), param.Manual); err != nil {
		return err
	} else {
		return ctx.String(http.StatusOK, name)
	}
}

func (r *Rest) GetChainGenesis(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	gsFile := path.Join(c.cfg.AbsBaseDir(), ChainGenesisZipFileName)
	return ctx.Attachment(gsFile, fmt.Sprintf("%s_%s", c.Channel(), ChainGenesisZipFileName))
}

func (r *Rest) GetChainConfig(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	return ctx.JSON(http.StatusOK, NewChainConfig(c.cfg))
}

func (r *Rest) ConfigureChain(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	p := &ConfigureParam{}
	if err := ctx.Bind(p); err != nil {
		return err
	}
	if err := r.n.ConfigureChain(c.CID(), p.Key, p.Value); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RunChainTask(ctx echo.Context) error {
	c := ctx.Get("chain").(*Chain)
	task := ctx.Param(TaskID)
	var params json.RawMessage
	if err := ctx.Bind(&params); err != nil {
		return err
	}
	if err := r.n.RunChainTask(c.CID(), task, params); err != nil {
		if errors.NotFoundError.Equals(err) {
			return ctx.String(http.StatusNotFound, fmt.Sprintf("%+v", err))
		}
		return err
	} else {
		return ctx.String(http.StatusOK, "OK")
	}
}

func (r *Rest) RegisterSystemHandlers(g *echo.Group) {
	g.GET("", r.GetSystem)
	g.GET("/configure", r.GetSystemConfig)
	g.POST("/configure", r.ConfigureSystem)
	r.RegistryBackupHandlers(g.Group("/backup"))
	r.RegistryRestoreHandlers(g.Group("/restore"))
}

func (r *Rest) GetSystem(ctx echo.Context) error {
	v := &SystemView{
		BuildVersion: r.n.cfg.BuildVersion,
		BuildTags:    r.n.cfg.BuildTags,
	}
	v.Setting.Address = r.n.w.Address().String()
	v.Setting.P2PAddr = r.n.nt.Address()
	v.Setting.P2PListenAddr = r.n.nt.GetListenAddress()
	v.Setting.RPCAddr = r.n.cfg.RPCAddr
	v.Setting.RPCDump = r.n.cfg.RPCDump
	v.Config = r.n.rcfg

	format := ctx.QueryParam("format")
	if format != "" {
		return defaultJsonTemplate.Response(format, v, ctx.Response())
	}
	return ctx.JSON(http.StatusOK, v)
}

func (r *Rest) GetSystemConfig(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, r.n.rcfg)
}

func (r *Rest) ConfigureSystem(ctx echo.Context) error {
	p := &ConfigureParam{}
	if err := ctx.Bind(p); err != nil {
		return err
	}

	if err := r.n.Configure(p.Key, p.Value); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RegistryBackupHandlers(g *echo.Group) {
	g.GET("", r.GetBackups)
}

func (r *Rest) GetBackups(ctx echo.Context) error {
	backups, err := r.n.GetBackups()
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, backups)
}

func (r *Rest) RegistryRestoreHandlers(g *echo.Group) {
	g.POST("", r.RestoreBackup)
	g.GET("", r.GetRestore)
	g.DELETE("", r.StopRestore)
}

func (r *Rest) GetRestore(ctx echo.Context) error {
	rv := r.n.GetRestore()
	return ctx.JSON(http.StatusOK, rv)
}

func (r *Rest) RestoreBackup(ctx echo.Context) error {
	param := new(RestoreBackupParam)
	if err := ctx.Bind(param); err != nil {
		return err
	}
	if err := r.n.StartRestore(param.Name, param.Overwrite); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) StopRestore(ctx echo.Context) error {
	if err := r.n.StopRestore(); err != nil {
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RegisterUserHandlers(g *echo.Group) {
	g.GET("", r.Users)
	g.POST("", r.AddUser)
	g.DELETE(UrlUserRes, r.RemoveUser)
}

func (r *Rest) Users(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, r.a.GetUsers())
}

func (r *Rest) AddUser(ctx echo.Context) error {
	param := struct {
		Id string `json:"id"`
	}{}
	if err := ctx.Bind(&param); err != nil {
		return echo.ErrBadRequest
	}
	if err := r.a.AddUser(param.Id); err != nil {
		if we, ok := err.(errors.Unwrapper); ok {
			switch we.Unwrap() {
			case ErrAlreadyExists:
				return ctx.String(http.StatusConflict, err.Error())
			}
		}
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RemoveUser(ctx echo.Context) error {
	p := ctx.Param(ParamID)
	if err := r.a.RemoveUser(p); err != nil {
		if errors.NotFoundError.Equals(err) {
			return ctx.String(http.StatusNotFound, err.Error())
		}
		return err
	}
	return ctx.String(http.StatusOK, "OK")
}

func (r *Rest) RegisterStatsHandlers(g *echo.Group) {
	g.GET("", r.StreamStats)
}

func (r *Rest) StreamStats(ctx echo.Context) error {
	intervalSec := 1
	param := ctx.QueryParam("interval")
	if param != "" {
		var err error
		intervalSec, err = strconv.Atoi(param)
		if err != nil {
			return err
		}
	}

	streaming := true
	param = ctx.QueryParam("stream")
	if param != "" {
		var err error
		streaming, err = strconv.ParseBool(param)
		if err != nil {
			return err
		}
	}
	// chains := ctx.QueryParam("chains")
	// strings.Split(chains,",")

	resp := ctx.Response()
	resp.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	resp.WriteHeader(http.StatusOK)
	if err := r.ResponseStatsView(resp); err != nil {
		return err
	}
	resp.Flush()

	tick := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer tick.Stop()
	for streaming {
		select {
		case <-tick.C:
			if err := r.ResponseStatsView(resp); err != nil {
				if EqualsSyscallErrno(err, syscall.EPIPE) {
					// ignore 'write: broken pipe' error
					// close by client
					return nil
				}
				return err
			}
			resp.Flush()
		}
	}
	return nil
}

func (r *Rest) ResponseStatsView(resp *echo.Response) error {
	v := StatsView{
		Chains:    make([]map[string]interface{}, 0),
		Timestamp: time.Now(),
	}
	for _, c := range r.n.GetChains() {
		m := metric.Inspect(c, false)
		if c.IsStarted() {
			m["cid"] = common.HexInt32{Value: int32(c.CID())}
			m["nid"] = common.HexInt32{Value: int32(c.NID())}
			m["channel"] = c.Channel()
			v.Chains = append(v.Chains, m)
		}
	}
	return json.NewEncoder(resp).Encode(&v)
}

func (r *Rest) RegisterDBHandlers(g *echo.Group) {
	bg := g.Group("/:"+ParamCID+"/:"+ParamBK, r.ChainInjector, r.BucketInjector)
	bg.GET("/:"+ParamKey, r.BucketGetValue)
}

func (r *Rest) BucketInjector(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		chain := ctx.Get("chain").(*Chain)
		bkID := db.BucketID(ctx.Param(ParamBK))
		var ret error
		chain.DoDBTask(func(database db.Database) {
			if database == nil {
				ret = ctx.String(http.StatusServiceUnavailable, "NoDatabase")
				return
			}
			bk, err := database.GetBucket(bkID)
			if err != nil {
				ret = ctx.String(http.StatusInternalServerError, "BucketFailure")
				return
			}
			ctx.Set("bucket", bk)
			ret = next(ctx)
			return
		})
		return ret
	}
}

func (r *Rest) BucketGetValue(ctx echo.Context) error {
	bk := ctx.Get("bucket").(db.Bucket)
	keyStr := ctx.Param(ParamKey)
	key, err := hex.DecodeString(keyStr)
	if err != nil {
		return ctx.String(http.StatusBadRequest, "InvalidKey(key:"+keyStr+")")
	}
	value, err := bk.Get(key)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	if value == nil {
		return ctx.JSON(http.StatusOK, nil)
	}
	return ctx.JSON(http.StatusOK, value)
}

func EqualsSyscallErrno(err error, sen syscall.Errno) bool {
	if oe, ok := err.(*net.OpError); ok {
		if se, ok := oe.Err.(*os.SyscallError); ok {
			if en, ok := se.Err.(syscall.Errno); ok && en == sen {
				return true
			}
		}
	}
	return false
}

type JsonTemplate struct {
	*template.Template
}

func NewJsonTemplate(name string) *JsonTemplate {
	tmpl := &JsonTemplate{template.New(name)}
	tmpl.Option("missingkey=error")
	tmpl.Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			a, _ := json.Marshal(v)
			return string(a)
		},
	})
	return tmpl
}

func (t *JsonTemplate) Response(format string, v interface{}, resp *echo.Response) error {
	nt, err := t.Clone()
	if err != nil {
		return err
	}
	nt, err = nt.Parse(format)
	if err != nil {
		return err
	}
	err = nt.Execute(resp, v)
	if err != nil {
		return err
	}

	// resp.Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	resp.Header().Set(echo.HeaderContentType, echo.MIMETextPlain)
	resp.WriteHeader(http.StatusOK)
	return nil
}
