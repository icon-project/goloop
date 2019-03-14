package rpc

import (
	"bytes"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/rpc/metric"
	"github.com/icon-project/goloop/rpc/v2"
	"github.com/icon-project/goloop/rpc/v3"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type JsonRpcServer struct {
	ch module.Chain
	bm module.BlockManager
	sm module.ServiceManager
	cs module.Consensus
	nm module.NetworkManager
}

func NewJsonRpcServer(ch module.Chain, bm module.BlockManager, sm module.ServiceManager, cs module.Consensus, nm module.NetworkManager) JsonRpcServer {
	return JsonRpcServer{
		ch: ch,
		bm: bm,
		sm: sm,
		cs: cs,
		nm: nm,
	}
}

func (s *JsonRpcServer) Start() error {

	log.Println("RPC - JsonRpcServer Start()")

	go func() {
		err := http.ListenAndServe(":9080", s.jsonRpcHandler())
		if err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}

func (s *JsonRpcServer) jsonRpcHandler() http.Handler {

	router := mux.NewRouter()

	// api
	router.Handle("/api/v2", &chunkHandler{v2.MethodRepository(s.bm, s.sm)})
	router.Handle("/api/v3", &chunkHandler{v3.MethodRepository(s.ch, s.bm, s.sm, s.cs)})

	corsOrigins := handlers.AllowedOrigins([]string{"*"})
	corsMethods := handlers.AllowedMethods([]string{"POST", "GET", "OPTIONS", "PUT", "DELETE"})
	corsHeaders := handlers.AllowedHeaders([]string{"Origin", "Accept", "X-Requested-With", "Content-Type", "Access-Control-Request-Method", "Access-Control-Allow-Headers", "Authorization"})
	maxAge := handlers.MaxAge(3600)

	// network
	if s.nm != nil {
		nmr := network.MethodRepository(s.nm)
		router.Handle("/network", handlers.CORS(corsOrigins, corsMethods, corsHeaders, maxAge)(nmr))
		router.PathPrefix("/view/network/").Handler(http.StripPrefix("/view/network/", &staticHandler{dir: "./html"}))
	}

	router.Handle("/status", statusMethodRepository(s.cs))
	router.Handle("/metrics", metric.PromethusExporter())
	// jaegerExporter()

	return router
}

type chunkHandler struct {
	next http.Handler
}

func (h *chunkHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// refer transfer.go 565
	if len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked" {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		rd := bytes.NewReader(b)
		r.ContentLength = int64(len(b))
		r.Body = ioutil.NopCloser(rd)
	}
	h.next.ServeHTTP(w, r)
}

type staticHandler struct {
	dir string
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if strings.HasSuffix(upath, "/") {
		upath = upath[:len(upath)-1]
	}
	i := strings.LastIndex(upath, "/")
	if i < 0 {
		upath = "/" + upath
		r.URL.Path = upath
	} else if i > 0 {
		http.Error(w, "invalid URL path!", http.StatusBadRequest)
		return
	}
	upath = h.dir + upath + ".html"
	// log.Println("staticHandler", upath)
	http.ServeFile(w, r, path.Clean(upath))
}

func (s *JsonRpcServer) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	errLogger := log.New(os.Stderr, "RPC|", log.Lshortfile|log.Lmicroseconds)
	srv := &http.Server{Handler: s.jsonRpcHandler(), ErrorLog: errLogger}
	return srv.Serve(ln)
}
