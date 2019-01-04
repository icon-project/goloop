package eeproxy

import (
	"log"
	"math/big"
	"sync"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
	"github.com/pkg/errors"
)

type Message uint

const (
	msgVESION     uint = 0
	msgINVOKE          = 1
	msgRESULT          = 2
	msgGETVALUE        = 3
	msgSETVALUE        = 4
	msgCALL            = 5
	msgEVENT           = 6
	msgGETINFO         = 7
	msgGETBALANCE      = 8
	msgGETAPI          = 9
)

type CallContext interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key, value []byte) error
	DeleteValue(key []byte) error
	GetInfo() *codec.TypedObj
	GetBalance(addr module.Address) *big.Int
	OnEvent(addr module.Address, indexed, data [][]byte)
	OnResult(status uint16, steps *big.Int, result *codec.TypedObj)
	OnCall(from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj)
	OnAPI(obj *scoreapi.Info)
}

type Proxy interface {
	Invoke(ctx CallContext, code string, isQuery bool, from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) error
	SendResult(ctx CallContext, status uint16, steps *big.Int, result *codec.TypedObj) error
	GetAPI(ctx CallContext, code string) error
	Release()
}

type callFrame struct {
	addr module.Address
	ctx  CallContext

	prev *callFrame
}

type proxy struct {
	lock     sync.Mutex
	reserved bool
	mgr      *manager

	conn ipc.Connection

	version   uint16
	pid       uint32
	scoreType scoreType

	frame *callFrame

	next  *proxy
	pprev **proxy
}

type versionMessage struct {
	Version uint16 `codec:"version"`
	PID     uint32 `codec:"pid"`
	Type    string
}

type invokeMessage struct {
	Code   string `codec:"code"`
	IsQry  bool
	From   common.Address  `codec:"from"`
	To     common.Address  `codec:"to"`
	Value  common.HexInt   `codec:"value"`
	Limit  common.HexInt   `codec:"limit"`
	Method string          `codec:"method"`
	Params *codec.TypedObj `codec:"params"`
}

type getValueMessage struct {
	Success bool
	Value   []byte
}

type setValueMessage struct {
	Key      []byte `codec:"key"`
	IsDelete bool
	Value    []byte `codec:"value"`
}

type callMessage struct {
	To     common.Address
	Value  common.HexInt
	Limit  common.HexInt
	Method string
	Params *codec.TypedObj
}

type eventMessage struct {
	Indexed [][]byte
	Data    [][]byte
}

func (p *proxy) Invoke(ctx CallContext, code string, isQuery bool, from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj) error {
	var m invokeMessage
	m.Code = code
	m.IsQry = isQuery
	m.From.SetBytes(from.Bytes())
	m.To.SetBytes(to.Bytes())
	m.Value.Set(value)
	m.Limit.Set(limit)
	m.Method = method
	m.Params = params

	p.lock.Lock()
	defer p.lock.Unlock()
	p.frame = &callFrame{
		addr: to,
		ctx:  ctx,
		prev: p.frame,
	}
	return p.conn.Send(msgINVOKE, &m)
}

func (p *proxy) GetAPI(ctx CallContext, code string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.frame = &callFrame{
		addr: nil,
		ctx:  ctx,
		prev: p.frame,
	}
	return p.conn.Send(msgGETAPI, code)
}

type resultMessage struct {
	Status   uint16
	StepUsed common.HexInt
	Result   *codec.TypedObj
}

func (p *proxy) reserve() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.reserved {
		return false
	}
	return true
}

func (p *proxy) Release() {
	p.lock.Lock()
	if !p.reserved {
		p.lock.Unlock()
		return
	}
	p.reserved = false
	if p.frame == nil {
		p.lock.Unlock()
		p.mgr.onReady(p.scoreType, p)
		return
	}
	p.lock.Unlock()
}

func (p *proxy) SendResult(ctx CallContext, status uint16, steps *big.Int, result *codec.TypedObj) error {
	var m resultMessage
	m.Status = status
	m.StepUsed.Set(steps)
	m.Result = result
	return p.conn.Send(msgRESULT, &m)
}

func (p *proxy) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	switch msg {
	case msgVESION:
		var m versionMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		p.version = m.Version
		p.pid = m.PID
		if t, ok := scoreNameToType[m.Type]; !ok {
			return errors.Errorf("UnknownSCOREName(%s)", m.Type)
		} else {
			p.scoreType = t
		}

		p.mgr.onReady(p.scoreType, p)
		return nil

	case msgRESULT:
		var m resultMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		p.lock.Lock()
		frame := p.frame
		p.frame = frame.prev
		p.lock.Unlock()

		frame.ctx.OnResult(m.Status, &m.StepUsed.Int, m.Result)

		p.lock.Lock()
		if p.frame == nil && !p.reserved {
			p.lock.Unlock()
			p.mgr.onReady(p.scoreType, p)
		} else {
			p.lock.Unlock()
		}
		return nil

	case msgGETVALUE:
		var key []byte
		if _, err := codec.MP.UnmarshalFromBytes(data, &key); err != nil {
			c.Close()
			return err
		}
		var m getValueMessage
		if value, err := p.frame.ctx.GetValue(key); err != nil {
			return err
		} else {
			if value != nil {
				m.Success = true
				m.Value = value
			} else {
				m.Success = false
				m.Value = nil
			}
		}
		return p.conn.Send(msgGETVALUE, &m)

	case msgSETVALUE:
		var m setValueMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		if m.IsDelete {
			return p.frame.ctx.DeleteValue(m.Key)
		} else {
			return p.frame.ctx.SetValue(m.Key, m.Value)
		}

	case msgCALL:
		var m callMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		p.frame.ctx.OnCall(p.frame.addr,
			&m.To, &m.Value.Int, &m.Limit.Int, m.Method, m.Params)
		return nil

	case msgEVENT:
		var m eventMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			c.Close()
			return err
		}
		p.frame.ctx.OnEvent(p.frame.addr, m.Indexed, m.Data)
		return nil

	case msgGETINFO:
		v := p.frame.ctx.GetInfo()
		eo, err := common.EncodeAny(v)
		if err != nil {
			return err
		}
		return p.conn.Send(msgGETINFO, eo)

	case msgGETBALANCE:
		var addr common.Address
		if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
			return err
		}
		var balance common.HexInt
		balance.Set(p.frame.ctx.GetBalance(&addr))
		return p.conn.Send(msgGETBALANCE, &balance)

	case msgGETAPI:
		var obj *scoreapi.Info
		if _, err := codec.MP.UnmarshalFromBytes(data, &obj); err != nil {
			return err
		} else {
			p.lock.Lock()
			frame := p.frame
			p.frame = frame.prev
			p.lock.Unlock()
			frame.ctx.OnAPI(obj)
			return nil
		}
	default:
		return errors.Errorf("UnknownMessage(%d)", msg)
	}
}

func (p *proxy) HandleMessages() error {
	for {
		err := p.conn.HandleMessage()
		if err != nil {
			log.Printf("Error on conn.HandleMessage() err=%+v\n", err)
			break
		}
	}
	p.mgr.detach(p)
	p.conn.Close()
	return nil
}

func newConnection(m *manager, c ipc.Connection) (*proxy, error) {
	p := &proxy{
		mgr:  m,
		conn: c,
	}
	c.SetHandler(msgVESION, p)
	c.SetHandler(msgRESULT, p)
	c.SetHandler(msgGETVALUE, p)
	c.SetHandler(msgSETVALUE, p)
	c.SetHandler(msgCALL, p)
	c.SetHandler(msgEVENT, p)
	c.SetHandler(msgGETINFO, p)
	c.SetHandler(msgGETBALANCE, p)
	c.SetHandler(msgGETAPI, p)
	return p, nil
}
