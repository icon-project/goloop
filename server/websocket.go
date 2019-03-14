package server

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
)

func Upgrader() *websocket.Upgrader {
	return &websocket.Upgrader{}
}

func wsEcho(c echo.Context) error {

	upgrader := Upgrader()
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	for {
		// Write
		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, Client!"))
		if err != nil {
			c.Logger().Error(err)
		}

		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			c.Logger().Error(err)
		}
		log.Printf("%s\n", msg)
	}
}
