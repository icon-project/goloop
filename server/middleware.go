package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/server/jsonrpc"
)

// JsonRpc()
func JsonRpc(mr *jsonrpc.MethodRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := new(jsonrpc.Request)
			if err := c.Bind(r); err != nil {
				return jsonrpc.ErrParse()
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
			return next(c)
		}
	}
}

func AnyChainInjector(srv *Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			c := srv.AnyChain()
			if c == nil {
				return jsonrpc.ErrorCodeServer.New("not exists chain")
			}
			ctx.Set("chain", c)
			return next(ctx)
		}
	}
}

func ChainInjector(srv *Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			channel := ctx.Param("channel")
			c := srv.Chain(channel)
			if c == nil {
				return ctx.NoContent(http.StatusNotFound)
			}
			ctx.Set("chain", c)
			return next(ctx)
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
