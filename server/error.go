package server

import (
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/labstack/echo/v4"
)

func HTTPErrorHandler(err error, c echo.Context) {
	if je, ok := err.(*jsonrpc.Error); ok {
		jsonrpc.ErrorHandler(je, c)
		return
	}
	c.Echo().DefaultHTTPErrorHandler(err, c)
}
