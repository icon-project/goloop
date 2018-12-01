package ipc

import (
	"github.com/icon-project/goloop/common/codec"
	codec2 "github.com/ugorji/go/codec"
	"log"
	"net"
	"sync"
)

type MessageHandler interface {
	HandleMessage(c Connection, msg uint, data []byte) error
}

type Connection interface {
	Send(msg uint, data interface{}) error
	SendAndReceive(msg uint, data interface{}, buf interface{}) error
	SetHandler(msg uint, handler MessageHandler)
	HandleMessage() error
	Close() error
}

type ConnectionHandler interface {
	OnConnect(c Connection) error
	OnClose(c Connection) error
}

type connection struct {
	lock    sync.Mutex
	conn    net.Conn
	handler map[uint]MessageHandler
}

type messageToSend struct {
	Msg  uint
	Data interface{}
}

func connectionFromConn(conn net.Conn) *connection {
	c := &connection{
		conn:    conn,
		handler: map[uint]MessageHandler{},
	}
	return c
}

func (c *connection) Send(msg uint, data interface{}) error {
	var m = messageToSend{
		Msg:  msg,
		Data: data,
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	return codec.MP.Marshal(c.conn, m)
}

type messageToReceive struct {
	Msg  uint
	Data codec2.Raw
}

func (c *connection) SendAndReceive(msg uint, data interface{}, buffer interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	var m = messageToSend{
		Msg:  msg,
		Data: data,
	}

	err := codec.MP.Marshal(c.conn, m)
	if err != nil {
		return err
	}

	var m2 messageToReceive
	if err := codec.MP.Unmarshal(c.conn, &m2); err != nil {
		return err
	}
	if _, err := codec.MP.UnmarshalFromBytes(m2.Data, buffer); err != nil {
		return err
	}
	return nil
}

func (c *connection) HandleMessage() error {
	var m messageToReceive
	if err := codec.MP.Unmarshal(c.conn, &m); err != nil {
		return err
	}
	c.lock.Lock()

	handler := c.handler[m.Msg]
	c.lock.Unlock()

	if handler == nil {
		log.Printf("Unknown message msg=%d\n", m.Msg)
		return nil
	}

	return handler.HandleMessage(c, m.Msg, m.Data)
}

func (c *connection) SetHandler(msg uint, handler MessageHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.handler[msg] = handler
}

func (c *connection) Close() error {
	return c.conn.Close()
}

func Dial(network, address string) (Connection, error) {
	if conn, err := net.Dial(network, address); err != nil {
		return nil, err
	} else {
		return connectionFromConn(conn), nil
	}
}
