package main

import (
	"log"
	"net/http"

	"github.com/icon-project/goloop/rpc"
)

func main()  {
	// JSON-RPC API serve
	err := http.ListenAndServe(":8080", rpc.JsonRpcHandler())
	if err != nil {
		log.Fatal(err)
	}
}
