package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/server/jsonrpc"
)

// JsonRpc()
func JsonRpc() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctype := c.Request().Header.Get(echo.HeaderContentType)
			if !strings.HasPrefix(ctype, echo.MIMEApplicationJSON) {
				c.Request().Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			}
			var raw json.RawMessage
			if err := c.Bind(&raw); err != nil {
				return jsonrpc.ErrParse()
			}
			c.Set("raw", raw)
			return next(c)
		}
	}
}

func ChainInjector(srv *Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			channel := ctx.Param("channel")
			c := srv.Chain(channel)
			if c == nil {
				return ctx.String(http.StatusNotFound, "No channel")
			}
			ctx.Set("chain", c)
			return next(ctx)
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

func NoneMiddlewareFunc(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}


func Unauthorized(readOnly bool) echo.MiddlewareFunc {
	if readOnly {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(ctx echo.Context) error {
				return ctx.String(http.StatusUnauthorized,"unauthorized")
			}
		}
	} else {
		return NoneMiddlewareFunc
	}
}
