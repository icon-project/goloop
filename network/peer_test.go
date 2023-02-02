package network

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/icon-project/goloop/common/log"
)

func testLogger() log.Logger {
	l := log.New()
	if testing.Verbose() {
		l.SetLevel(log.TraceLevel)
		l.SetConsoleLevel(log.TraceLevel)
	}
	return l
}

type fakeConn struct {
	net.Conn
	readBuf  *list.List
	writeBuf *bytes.Buffer
	ch       chan *Packet
	mtx      sync.RWMutex

	errClose            error
	errRead             error
	errWrite            error
	errSetDeadline      error
	errSetReadDeadline  error
	errSetWriteDeadline error
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		readBuf:  list.New(),
		writeBuf: bytes.NewBuffer(make([]byte, 0, DefaultPacketPayloadMax)),
		ch:       make(chan *Packet, 100),
	}
}

func (c *fakeConn) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.errClose != nil {
		return c.errClose
	}

	if c.ch == nil {
		return net.ErrClosed
	}
	close(c.ch)
	c.ch = nil
	return nil
}
func (c *fakeConn) IsClosed() bool {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.ch == nil
}
func (c *fakeConn) read() <-chan *Packet {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	return c.ch
}
func (c *fakeConn) Read(p []byte) (int, error) {
	if c.errRead != nil {
		return 0, c.errRead
	}

	if c.IsClosed() {
		return 0, net.ErrClosed
	}
	if c.writeBuf.Len() > 0 {
		n, err := c.writeBuf.Read(p)
		if err != nil {
			return 0, err
		}
		if c.writeBuf.Len() == 0 {
			c.writeBuf.Reset()
		}
		return n, nil
	}
	for {
		ch := c.read()
		select {
		case pkt := <-ch:
			if pkt == nil {
				return 0, io.EOF
			}
			if _, err := pkt.WriteTo(c.writeBuf); err != nil {
				return 0, err
			}
			return c.writeBuf.Read(p)
		}
	}
}
func (c *fakeConn) Write(p []byte) (int, error) {
	if c.errWrite != nil {
		return 0, c.errWrite
	}
	pkt := &Packet{}
	n, err := pkt.ReadFrom(bytes.NewReader(p))
	c.readBuf.PushBack(pkt)
	return int(n), err
}
func (c *fakeConn) LocalAddr() net.Addr                { return nil }
func (c *fakeConn) RemoteAddr() net.Addr               { return nil }
func (c *fakeConn) SetDeadline(t time.Time) error      { return c.errSetDeadline }
func (c *fakeConn) SetReadDeadline(t time.Time) error  { return c.errSetReadDeadline }
func (c *fakeConn) SetWriteDeadline(t time.Time) error { return c.errSetWriteDeadline }
func (c *fakeConn) Packet() *Packet {
	if c.readBuf.Len() > 0 {
		if p, ok := c.readBuf.Remove(c.readBuf.Front()).(*Packet); ok {
			return p
		}
	}
	return nil
}
func (c *fakeConn) WritePacket(pkt *Packet) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	select {
	case c.ch <- pkt:
		return nil
	default:
		fmt.Println("fakeConn.WritePacket fail")
		return net.ErrClosed
	}
}

func newPeerWithFakeConn(in bool) (*Peer, *fakeConn) {
	conn := newFakeConn()
	return newPeer(conn, in, "", testLogger()), conn
}

func Test_PeerRTT(t *testing.T) {
	sleepTime := 100 * time.Millisecond
	r := NewPeerRTT()
	r.Start()
	time.Sleep(sleepTime)
	actual := r.Stop()
	expected := r.et.Sub(r.st)
	assert.Equal(t, expected, actual)
	last, avg := r.Value()
	assert.Equal(t, expected, last)
	assert.Equal(t, expected, avg)
	converted := float64(expected) / float64(time.Millisecond)
	assert.Equal(t, converted, r.Last(time.Millisecond))
	assert.Equal(t, converted, r.Avg(time.Millisecond))
	t.Log(r.String())

	wg := sync.WaitGroup{}
	wg.Add(1)
	r.StartWithAfterFunc(sleepTime-time.Millisecond, func() {
		wg.Done()
	})
	time.Sleep(sleepTime)
	actual = r.Stop()
	timer := time.AfterFunc(time.Second, func() {
		assert.FailNow(t, "timeout")
	})
	wg.Wait()
	timer.Stop()
	assert.Equal(t, r.et.Sub(r.st), actual)
	//exponential weighted moving average model
	//avg = (1-0.125)*avg + 0.125*last
	fv := 0.875*float64(avg) + 0.125*float64(actual)
	_, avg = r.Value()
	assert.Equal(t, time.Duration(fv), avg)
	t.Log(r.String())
}

func Test_PeerRoleFlag(t *testing.T) {
	pr := p2pRoleNone
	assert.False(t, pr.Has(p2pRoleSeed))
	assert.False(t, pr.Has(p2pRoleRoot))
	assert.Equal(t, p2pRoleNone, pr)

	pr.SetFlag(p2pRoleSeed)
	assert.True(t, pr.Has(p2pRoleSeed))
	assert.False(t, pr.Has(p2pRoleRoot))
	assert.Equal(t, p2pRoleSeed, pr)

	pr.SetFlag(p2pRoleRoot)
	assert.True(t, pr.Has(p2pRoleSeed))
	assert.True(t, pr.Has(p2pRoleRoot))

	assert.Equal(t, p2pRoleSeed|p2pRoleRoot, pr)
}
