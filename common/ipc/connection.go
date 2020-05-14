package ipc

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/icon-project/goloop/common/codec"
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
	OnClose(c Connection)
}

type connection struct {
	lock    sync.Mutex
	conn    net.Conn
	reader  io.Reader
	handler map[uint]MessageHandler
	closed  bool
}

type messageToSend struct {
	Msg  uint
	Data interface{}
}

func connectionFromConn(conn net.Conn) *connection {
	c := &connection{
		conn:    conn,
		reader:  bufio.NewReader(conn),
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

type rawMessage []byte

func (m *rawMessage) UnmarshalRLP(bs []byte) error {
	n := make([]byte, len(bs))
	copy(n, bs)
	*m = n
	return nil
}

type messageToReceive struct {
	Msg  uint
	Data rawMessage
}

func (m *messageToReceive) RawData() []byte {
	return m.Data
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
	if err := codec.MP.Unmarshal(c.reader, &m2); err != nil {
		return err
	}
	if _, err := codec.MP.UnmarshalFromBytes(m2.RawData(), buffer); err != nil {
		return err
	}
	return nil
}

func (c *connection) getHandler(msg uint) MessageHandler {
	c.lock.Lock()
	defer c.lock.Unlock()
	if handler, ok := c.handler[msg]; ok {
		return handler
	}
	return nil
}

func (c *connection) HandleMessage() error {
	var m messageToReceive
	if err := codec.MP.Unmarshal(c.reader, &m); err != nil {
		if c.closed {
			return io.EOF
		}
		return err
	}

	handler := c.getHandler(m.Msg)
	if handler == nil {
		return fmt.Errorf("UnknownMessage(msg=%d)", m.Msg)
	}

	return handler.HandleMessage(c, m.Msg, m.RawData())
}

func (c *connection) SetHandler(msg uint, handler MessageHandler) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if handler == nil {
		delete(c.handler, msg)
		return
	}
	c.handler[msg] = handler
}

func (c *connection) Close() error {
	c.closed = true
	return c.conn.Close()
}

func Dial(network, address string) (Connection, error) {
	if conn, err := net.Dial(network, address); err != nil {
		return nil, err
	} else {
		return connectionFromConn(conn), nil
	}
}
