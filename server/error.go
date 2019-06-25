package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/server/jsonrpc"
)

func HTTPErrorHandler(err error, c echo.Context) {
	if je, ok := err.(*jsonrpc.Error); ok {
		jsonrpc.ErrorHandler(je, c)
		return
	}
	if he, ok := err.(*echo.HTTPError); ok {
		err = c.String(he.Code, fmt.Sprintf("%s %+v", he.Message, he.Internal))
	} else {
		err = c.String(http.StatusInternalServerError,
			fmt.Sprintf("%+v", err))
	}
	if err != nil {
		c.Echo().DefaultHTTPErrorHandler(err, c)
	}
}
