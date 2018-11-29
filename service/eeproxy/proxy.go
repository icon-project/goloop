package eeproxy

import (
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/module"
	"github.com/pkg/errors"
	"log"
	"math/big"
)

const (
	M_VERSION  uint = 0
	M_INVOKE        = 1
	M_RESULT        = 2
	M_GETVALUE      = 3
	M_SETVALUE      = 4
	M_CALL          = 5
	M_EVENT         = 6
)

type CallContext interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key, value []byte) error
	OnEvent(idxcnt uint16, msgs [][]byte)
	OnResult(code uint16, steps *big.Int)
	OnCall(from, to module.Address, value, limit *big.Int)
}

type Proxy interface {
	Invoke(ctx CallContext, code string, from, to module.Address,
		value, limit *big.Int, method string, params interface{}) error
	SendResult(ctx CallContext, status uint16, steps *big.Int) error
}

type callFrame struct {
	addr module.Address
	ctx  CallContext

	prev *callFrame
}

type proxy struct {
	conn ipc.Connection

	frame *callFrame

	next  *proxy
	pprev **proxy
}

type versionMessage struct {
	Version uint16 `codec:"version"`
	PID     uint32 `codec:"pid"`
}

type invokeMessage struct {
	Code   string         `codec:"code"`
	From   common.Address `codec:"from"`
	To     common.Address `codec:"to"`
	Value  common.HexInt  `codec:"value"`
	Limit  common.HexInt  `codec:"limit"`
	Method string         `codec:"method"`
	Params interface{}    `codec:"params"`
}

type resultMessage struct {
	status   uint16        `codec:"status"`
	stepUsed common.HexInt `codec:"stepUsed"`
}

type setMessage struct {
	Key   []byte `codec:"key"`
	Value []byte `codec:"value"`
}

type callMessage struct {
	To     common.Address
	Value  common.HexInt
	Limit  common.HexInt
	Method string
	Params interface{}
}

type eventMessage struct {
	Index    uint16
	Messages [][]byte
}

func (p *proxy) Invoke(ctx CallContext, code string, from, to module.Address,
	value, limit *big.Int, method string, params interface{},
) error {
	var m invokeMessage
	m.Code = code
	m.From.SetBytes(from.Bytes())
	m.To.SetBytes(to.Bytes())
	m.Value.Set(value)
	m.Limit.Set(limit)
	m.Method = method
	m.Params = params

	p.frame = &callFrame{
		addr: to,
		ctx:  ctx,
		prev: p.frame,
	}
	return p.conn.Send(M_INVOKE, &m)
}

func (p *proxy) SendResult(ctx CallContext, status uint16, stepUsed *big.Int) error {
	var m resultMessage
	m.status = status
	m.stepUsed.Set(stepUsed)
	return p.conn.Send(M_RESULT, &m)
}

func (p *proxy) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	switch msg {
	case M_VERSION:
		var m versionMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		log.Println("VERSION:%d, PID:%d", m.Version, m.PID)
		return nil

	case M_CALL:
		var m callMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		p.frame.ctx.OnCall(p.frame.addr, &m.To, &m.Value.Int, &m.Limit.Int)
		return nil

	case M_RESULT:
		var m resultMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		frame := p.frame
		p.frame = frame.prev
		frame.ctx.OnResult(m.status, &m.stepUsed.Int)
		return nil

	case M_GETVALUE:
		var m []byte
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		value, err := p.frame.ctx.GetValue(m)
		if err != nil || value == nil {
			value = []byte{}
		}
		return p.conn.Send(M_GETVALUE, value)

	case M_SETVALUE:
		var m setMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		return p.frame.ctx.SetValue(m.Key, m.Value)

	case M_EVENT:
		var m eventMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		p.frame.ctx.OnEvent(m.Index, m.Messages)
		return nil

	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
}

func newConnection(c ipc.Connection) (*proxy, error) {
	p := &proxy{
		conn: c,
	}
	c.SetHandler(M_VERSION, p)
	c.SetHandler(M_RESULT, p)
	c.SetHandler(M_GETVALUE, p)
	c.SetHandler(M_SETVALUE, p)
	c.SetHandler(M_CALL, p)
	return p, nil
}
