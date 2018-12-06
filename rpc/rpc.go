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

	"github.com/gorilla/mux"
)

type JsonRpcServer struct {
	bm module.BlockManager
	sm module.ServiceManager
	nt module.NetworkTransport
}

func NewJsonRpcServer(bm module.BlockManager, sm module.ServiceManager, nt module.NetworkTransport) JsonRpcServer {
	return JsonRpcServer{
		bm: bm,
		sm: sm,
		nt: nt,
	}
}

func (s *JsonRpcServer) Start() error {

	log.Println("RPC - JsonRpcServer Start()")

	go func() {
		err := http.ListenAndServe(":8080", s.jsonRpcHandler())
		if err != nil {
			log.Fatal(err)
		}
	}()

	return nil
}

func (s *JsonRpcServer) jsonRpcHandler() http.Handler {

	router := mux.NewRouter()

	v2 := v2.MethodRepository(s.bm, s.sm)
	v3 := v3.MethodRepository(s.bm, s.sm)

	router.Handle("/api/v2", v2)
	router.Handle("/api/v3", v3)

	if s.nt != nil {
		nmr := network.MethodRepository(s.nt)
		router.Handle("/network", &corsHandler{next: nmr})
		dir := "./network"
		// if ex, err := os.Executable(); err == nil {
		// 	dir = filepath.Dir(ex)
		// }
		router.PathPrefix("/static/network/").Handler(&corsHandler{http.StripPrefix("/static/network/", &staticHandler{dir: dir})})
	}
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

type corsHandler struct {
	next http.Handler
}

func (h *corsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.next.ServeHTTP(w, req)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Max-Age", "3600")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Accept, X-Requested-With, Content-Type, Access-Control-Request-Method, Access-Control-Allow-Headers, Authorization")
}

func (s *JsonRpcServer) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.jsonRpcHandler())
}
