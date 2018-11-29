package ipc

import (
	"github.com/pkg/errors"
	"testing"
	"time"
)

type testConnectionHandler struct {
	onConnect, onClose           int
	errorOnConnect, errorOnClose bool
}

func (ch *testConnectionHandler) OnConnect(c Connection) error {
	ch.onConnect++
	if ch.errorOnConnect {
		return errors.New("ErrorOnConnectByIntention")
	}
	return nil
}

func (ch *testConnectionHandler) OnClose(c Connection) error {
	ch.onClose++
	if ch.errorOnClose {
		return errors.New("ErrorOnCloseByIntention")
	}
	return nil
}

func TestServer_Close(t *testing.T) {
	s := NewServer()
	if err := s.Listen("unix", "/tmp/test2"); err != nil {
		t.Errorf("Fail to listen err=%+v", err)
		return
	}

	handler := testConnectionHandler{}
	s.SetHandler(&handler)

	loopEnded := false
	go func() {
		s.Loop()
		loopEnded = true
	}()

	time.Sleep(100 * time.Millisecond)

	if loopEnded {
		t.Errorf("Loop is already ended")
		return
	}

	s.Close()

	time.Sleep(100 * time.Millisecond)

	if !loopEnded {
		t.Errorf("Loop isn't ended on close")
		return
	}
}

func TestServer_SetHandler(t *testing.T) {
	n, a := "unix", "/tmp/test2"
	s := NewServer()
	if err := s.Listen(n, a); err != nil {
		t.Errorf("Fail to listen err=%+v", err)
		return
	}

	handler := testConnectionHandler{}
	s.SetHandler(&handler)
	go s.Loop()

	time.Sleep(100 * time.Millisecond)

	go func() {
		conn, err := Dial(n, a)
		if err != nil {
			t.Errorf("Fail to dial err=%+v", err)
			return
		}

		time.Sleep(100 * time.Millisecond)

		if handler.onConnect != 1 {
			t.Errorf("Invalid state connection isn't notified")
			return
		}

		conn.Close()
	}()

	time.Sleep(200 * time.Millisecond)

	if handler.onClose != 1 || handler.onConnect != 1 {
		t.Errorf("Invalid state conn=%d close=%d",
			handler.onConnect, handler.onClose)
		return
	}

	handler.errorOnConnect = true

	go func() {
		conn, err := Dial(n, a)
		if err != nil {
			t.Errorf("Fail to dial err=%+v", err)
			return
		}

		time.Sleep(100 * time.Millisecond)

		if handler.onConnect != 2 {
			t.Errorf("Invalid state connection isn't notified")
			return
		}

		err = conn.Send(1, []byte{0x01})
		if err == nil {
			t.Errorf("Connection should not be available.")
			return
		}

		conn.Close()
	}()

	time.Sleep(200 * time.Millisecond)

	if handler.onClose != 1 || handler.onConnect != 2 {
		t.Errorf("Invalid state")
	}

	s.Close()
}
