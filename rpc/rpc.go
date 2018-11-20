package rpc

import (
	"log"
	"net/http"

	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/rpc/v2"
	"github.com/icon-project/goloop/rpc/v3"

	"github.com/gorilla/mux"
)

type JsonRpcServer struct {
	bm module.BlockManager
	sm module.ServiceManager
}

func NewJsonRpcServer(bm module.BlockManager, sm module.ServiceManager) JsonRpcServer {
	return JsonRpcServer{
		bm: bm,
		sm: sm,
	}
}

func (s *JsonRpcServer) Start() error {

	err := http.ListenAndServe(":8080", s.jsonRpcHandler())
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func (s *JsonRpcServer) jsonRpcHandler() http.Handler {

	router := mux.NewRouter()

	v2 := v2.MethodRepository(s.bm, s.sm)
	v3 := v3.MethodRepository(s.bm, s.sm)

	router.Handle("/api/v2", v2)
	router.Handle("/api/v3", v3)

	return router
}
