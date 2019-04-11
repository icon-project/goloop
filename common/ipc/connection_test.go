// +build ignore

package ipc

import (
	"github.com/icon-project/goloop/common/codec"
	"log"
	"testing"
)

type msgHandlerHello struct {
}

func (mh *msgHandlerHello) HandleMessage(c Connection, msg uint, data []byte) error {
	var buf []byte
	_, err := codec.MP.UnmarshalFromBytes(data, &buf)
	if err != nil {
		log.Printf("Fail to unmarshal bytes:% X", data)
	}
	log.Printf("MsgHandlerHello data:%s", string(buf))
	return c.Send(msg, "hello")
}

type connHandler struct {
}

func (ch *connHandler) OnConnect(c Connection) error {
	log.Printf("OnConnect() conn=%+v", c)
	c.SetHandler(1, &msgHandlerHello{})
	return nil
}

func (ch *connHandler) OnClose(c Connection) error {
	return nil
}

func Test_server_self_connection(t *testing.T) {
	domain, addr := "unix", "/tmp/test"
	server := NewServer()
	server.SetHandler(&connHandler{})
	server.Listen(domain, addr)
	go server.Loop()

	conn, err := Dial(domain, addr)
	if err != nil {
		t.Errorf("Fail to dial %s:%s err=%+v", domain, addr, err)
	}

	var buf string
	conn.SendAndReceive(1, []byte("TEST"), &buf)
	log.Printf("Result:%s", buf)
}
