package server

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/labstack/echo"

	"github.com/icon-project/goloop/server/jsonrpc"
)

// JsonRpc()
func JsonRpc(srv *Manager, mr *jsonrpc.MethodRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := new(jsonrpc.Request)
			if err := c.Bind(r); err != nil {
				return jsonrpc.ErrInvalidRequest()
			}
			c.Set("request", r)
			if err := c.Validate(r); err != nil {
				return jsonrpc.ErrInvalidRequest()
			}
			method, err := mr.TakeMethod(r)
			if err != nil {
				return err
			}
			c.Set("method", method)

			// TODO : ChainManager.Chain(channel)
			// channel := c.Param("channel")
			c.Set("chain", *srv.Chain())

			return next(c)
		}
	}
}

// JsonRpcLogger()
func JsonRpcLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Get("request").(*jsonrpc.Request)
			method := req.Method
			// TODO : jsonrpc logging
			fmt.Printf("method: %s\n", method)
			return next(c)
		}
	}
}

// Chunk()
func Chunk() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			if len(r.TransferEncoding) > 0 && r.TransferEncoding[0] == "chunked" {
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				rd := bytes.NewReader(b)
				r.ContentLength = int64(len(b))
				r.Body = ioutil.NopCloser(rd)
			}
			return next(c)
		}
	}
}
