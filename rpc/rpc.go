package rpc

import (
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/network"
	"github.com/icon-project/goloop/rpc/v2"
	"github.com/icon-project/goloop/rpc/v3"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type JsonRpcServer struct {
	bm module.BlockManager
	sm module.ServiceManager
	cs module.Consensus
	nm module.NetworkManager
}

func NewJsonRpcServer(bm module.BlockManager, sm module.ServiceManager, cs module.Consensus, nm module.NetworkManager) JsonRpcServer {
	return JsonRpcServer{
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
	router.Handle("/api/v2", v2.MethodRepository(s.bm, s.sm))
	router.Handle("/api/v3", v3.MethodRepository(s.bm, s.sm))

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

	// status
	router.Handle("/status", statusMethodRepository(s.cs))
	router.Handle("/metrics", promethusExporter(s.cs))
	// jaegerExporter()

	return router
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
	return http.ListenAndServe(addr, s.jsonRpcHandler())
}
