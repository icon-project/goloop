package ipc

import (
	"log"
	"net"
	"os"
	"sync"
)

type Server interface {
	// Listen specified port to watch.
	Listen(net, addr string) error

	// SetHandler set handler for connection. The handler can add message
	// handler for the connection, and clean-up resource on close.
	SetHandler(handler ConnectionHandler)

	// Loop handles connection requests. If it sees I/O errors, it
	// automatically close port and return the error.
	Loop() error

	// Close the port, and it causes loop end.
	Close() error

	Addr() net.Addr
}

type server struct {
	mutex    sync.Mutex
	listener net.Listener
	handler  ConnectionHandler
}

func (s *server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *server) Listen(network, address string) error {
	switch network {
	case "unix":
		os.Remove(address)
	}
	listener, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	if ulsr, ok := listener.(*net.UnixListener); ok {
		ulsr.SetUnlinkOnClose(true)
	}
	s.listener = listener
	return nil
}

func (s *server) SetHandler(handler ConnectionHandler) {
	s.handler = handler
}

func (s *server) handleConnection(conn net.Conn) {
	co := connectionFromConn(conn)
	handler := s.handler
	if handler != nil {
		if err := handler.OnConnect(co); err != nil {
			log.Printf("Fail on OnConnect() err=%+v", err)
			co.Close()
			return
		}
	}

	for {
		err := co.HandleMessage()
		if err != nil {
			log.Printf("Fail to handle message err=%+v", err)
			break
		}
	}

	if handler != nil {
		if err := handler.OnClose(co); err != nil {
			log.Printf("Fail on OnClose() err=%+v", err)
		}
	}
	co.Close()
}

func (s *server) Loop() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(conn)
	}
	s.listener.Close()
	return nil
}

func (s *server) Close() error {
	return s.listener.Close()
}

func NewServer() Server {
	return new(server)
}
