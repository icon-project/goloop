package rpc

import (
	"net/http"

	"github.com/icon-project/goloop/rpc/v2"
	"github.com/icon-project/goloop/rpc/v3"

	"github.com/gorilla/mux"
)

func JsonRpcService() http.Handler {

	router := mux.NewRouter()

	v2 := v2.MethodRepository()
	v3 := v3.MethodRepository()

	router.Handle("/api/v2", v2)
	router.Handle("/api/v3", v3)

	return router
}
