package eeproxy

import (
	"math/big"
	"sync"

	"github.com/gofrs/uuid"

	"github.com/icon-project/goloop/common/log"
	"github.com/icon-project/goloop/service/trace"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/ipc"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/service/scoreapi"
)

type Message uint

const (
	msgVERSION     = 0
	msgINVOKE      = 1
	msgRESULT      = 2
	msgGETVALUE    = 3
	msgSETVALUE    = 4
	msgCALL        = 5
	msgEVENT       = 6
	msgGETINFO     = 7
	msgGETBALANCE  = 8
	msgGETAPI      = 9
	msgLOG         = 10
	msgCLOSE       = 11
	msgSETCODE     = 12
	msgGETOBJGRAPH = 13
	msgSETOBJGRAPH = 14
	msgSETFEEPCT   = 15
	msgCONTAINS    = 16
)

type proxyState int

const (
	stateIdle proxyState = iota
	stateReady
	stateReserved
	stateStopped
	stateClosed
)

const (
	ModuleName = "eeproxy"
)

type CodeState struct {
	NexHash   int
	GraphHash []byte
	PrevEID   int
}

type CallContext interface {
	GetValue(key []byte) ([]byte, error)
	SetValue(key []byte, value []byte) ([]byte, error)
	DeleteValue(key []byte) ([]byte, error)
	ArrayDBContains(prefix, value []byte, limit int64) (bool, int, int, error)
	GetInfo() *codec.TypedObj
	GetBalance(addr module.Address) *big.Int
	OnEvent(addr module.Address, indexed, data [][]byte) error
	OnResult(status error, flag int, steps *big.Int, result *codec.TypedObj)
	OnCall(from, to module.Address, value, limit *big.Int, dataType string, dataObj *codec.TypedObj)
	OnAPI(status error, info *scoreapi.Info)
	OnSetFeeProportion(portion int)
	SetCode(code []byte) error
	GetObjGraph(bool) (int, []byte, []byte, error)
	SetObjGraph(flags bool, nextHash int, objGraph []byte) error
	Logger() log.Logger
}

type Proxy interface {
	Invoke(ctx CallContext, code string, isQuery bool, from, to module.Address,
		value, limit *big.Int, method string, params *codec.TypedObj,
		cid []byte, eid int, state *CodeState) error
	SendResult(ctx CallContext, status error, steps *big.Int, result *codec.TypedObj, eid int, last int) error
	GetAPI(ctx CallContext, code string) error
	Release()
	Kill() error
}

type proxyManager interface {
	onReady(p *proxy) error
	kill(u string) error
}

type callFrame struct {
	addr module.Address
	ctx  CallContext
	log  *trace.Logger

	prev *callFrame
}

type proxy struct {
	lock  sync.Mutex
	state proxyState
	mgr   proxyManager

	conn ipc.Connection

	version   uint16
	uid       string
	scoreType string

	log *trace.Logger

	frame *callFrame

	next  *proxy
	pprev **proxy
}

type versionMessage struct {
	Version uint16 `codec:"version"`
	UID     string
	Type    string
}

type invokeFlag int

const (
	InvokeFlagReadOnly invokeFlag = 1 << iota
	InvokeFlagTrace
)

type invokeMessage struct {
	Code   string `codec:"code"`
	Flag   invokeFlag
	From   *common.Address `codec:"from"`
	To     common.Address  `codec:"to"`
	Value  common.HexInt   `codec:"value"`
	Limit  common.HexInt   `codec:"limit"`
	Method string          `codec:"method"`
	Params *codec.TypedObj `codec:"params"`
	Info   *codec.TypedObj `codec:"info"`
	CID    []byte
	EID    int
	State  *CodeState
}

type getValueMessage struct {
	Success bool
	Value   []byte
}

type setValueMessage struct {
	Key   []byte `codec:"key"`
	Flag  uint16
	Value []byte `codec:"value"`
}

const (
	flagDELETE uint16 = 1 << iota
	flagOLDVALUE
)

type oldValueMessage struct {
	HasOld  bool
	OldSize int
}

type callMessage struct {
	To       common.Address
	Value    common.HexInt
	Limit    common.HexInt
	DataType string
	Data     *codec.TypedObj
}

type eventMessage struct {
	Indexed [][]byte
	Data    [][]byte
}

type getAPIMessage struct {
	Status errors.Code
	Info   *scoreapi.Info
}

type logFlag int

const (
	LogFlagTrace logFlag = 1 << iota
)

type logMessage struct {
	Level   log.Level
	Flag    logFlag
	Message string
}

type getObjGraphMessage struct {
	NextHash    int
	GraphHash   []byte
	ObjectGraph []byte
}

type setObjGraphMessage struct {
	Flags       int
	NextHash    int
	ObjectGraph []byte
}

type containsMessage struct {
	Prefix []byte
	Value  []byte
	Limit  int64
}

type containsResponse struct {
	YN    bool
	Count int
	Size  int
}

func traceLevelOf(lv log.Level) module.TraceLevel {
	switch lv {
	case log.DebugLevel:
		return module.TDebugLevel
	case log.TraceLevel:
		return module.TTraceLevel
	default:
		return module.TSystemLevel
	}
}

func (p *proxy) Invoke(
	ctx CallContext, code string, isQuery bool,
	from, to module.Address, value, limit *big.Int, method string, params *codec.TypedObj,
	cid []byte, eid int, state *CodeState,
) error {
	logger := trace.LoggerOf(ctx.Logger().WithFields(log.Fields{log.FieldKeyEID: p.uid}))

	var m invokeMessage
	m.Code = code
	m.Flag = 0
	if isQuery {
		m.Flag |= InvokeFlagReadOnly
	}
	if logger.IsTrace() {
		m.Flag |= InvokeFlagTrace
	}
	m.From = common.AddressToPtr(from)
	m.To.Set(to)
	m.Value.Set(value)
	m.Limit.Set(limit)
	m.Method = method
	m.Params = params
	m.CID = cid
	m.EID = eid
	m.State = state

	v := ctx.GetInfo()
	if eo, err := common.EncodeAny(v); err != nil {
		return err
	} else {
		m.Info = eo
	}

	logger.Tracef("Proxy[%p].Invoke code=%s query=%v from=%v to=%v value=%v limit=%v method=%s eid=%d", p, code, isQuery, from, to, value, limit, method, eid)

	p.lock.Lock()
	defer p.lock.Unlock()
	p.frame = &callFrame{
		addr: to,
		ctx:  ctx,
		log:  p.log,
		prev: p.frame,
	}
	p.log = logger
	return p.conn.Send(msgINVOKE, &m)
}

func (p *proxy) GetAPI(ctx CallContext, code string) error {
	logger := trace.LoggerOf(ctx.Logger().WithFields(log.Fields{log.FieldKeyEID: p.uid}))

	logger.Tracef("Proxy[%p].GetAPI(code=%s)", p, code)

	p.lock.Lock()
	defer p.lock.Unlock()
	p.frame = &callFrame{
		addr: nil,
		ctx:  ctx,
		log:  p.log,
		prev: p.frame,
	}
	p.log = logger
	return p.conn.Send(msgGETAPI, code)
}

const (
	CodeBits = 24
	CodeMask = (1 << CodeBits) - 1
)

func StatusToCodeAndFlag(code errors.Code) (errors.Code, int) {
	return code & CodeMask, int(code >> CodeBits)
}

type resultMessage struct {
	Status   errors.Code
	StepUsed common.HexInt
	Result   *codec.TypedObj
	EID      int
	PrevEID  int
}

func (p *proxy) reserve() bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.state == stateReady {
		p.state = stateReserved
		return true
	}
	return false
}

func (p *proxy) Release() {
	l := common.LockForAutoCall(&p.lock)
	defer l.Unlock()
	if p.state != stateReserved {
		return
	}
	p.state = stateIdle
	if p.tryToBeReadyInLock() {
		l.CallAfterUnlock(func() {
			p.mgr.onReady(p)
		})
	}
}

func (p *proxy) SendResult(ctx CallContext, status error, steps *big.Int, result *codec.TypedObj, eid int, last int) error {
	p.log.Tracef("Proxy[%p].SendResult status=%v steps=%v last=%d eid=%d", p, status, steps, last, eid)
	var m resultMessage
	m.StepUsed.Set(steps)
	if status == nil {
		m.Status = errors.Success
		if result == nil {
			result = codec.Nil
		}
		m.Result = result
	} else {
		m.Status = errors.CodeOf(status)
		m.Result = common.MustEncodeAny(status.Error())
	}
	m.EID = eid
	m.PrevEID = last
	return p.conn.Send(msgRESULT, &m)
}

func (p *proxy) popFrame() *callFrame {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.frame != nil {
		frame := p.frame
		p.log = frame.log
		p.frame = frame.prev
		return frame
	}
	return nil
}

func (p *proxy) tryToBeReady() error {
	l := common.LockForAutoCall(&p.lock)
	defer l.Unlock()
	if p.state >= stateStopped {
		return common.ErrInvalidState
	}
	if p.tryToBeReadyInLock() {
		l.CallAfterUnlock(func() {
			p.mgr.onReady(p)
		})
	}
	return nil
}

func (p *proxy) tryToBeReadyInLock() bool {
	if p.frame == nil && p.state == stateIdle {
		p.state = stateReady
	}
	return p.state == stateReady
}

func (p *proxy) HandleMessage(c ipc.Connection, msg uint, data []byte) error {
	switch msg {
	case msgRESULT:
		var m resultMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		p.log.Tracef("Proxy[%p].OnResult status=%s steps=%v", p, module.Status(m.Status), &m.StepUsed.Int)

		frame := p.popFrame()
		if frame == nil {
			return errors.InvalidStateError.New("Empty frame")
		}

		var status error
		var result *codec.TypedObj
		var statusFlag int
		m.Status, statusFlag = StatusToCodeAndFlag(m.Status)
		if m.Status == errors.Success {
			status = nil
			result = m.Result
		} else {
			msg := common.DecodeAsString(m.Result, "")
			status = m.Status.New(msg)
			result = nil
		}
		frame.ctx.OnResult(status, statusFlag, &m.StepUsed.Int, result)

		return p.tryToBeReady()
	case msgGETVALUE:
		var key []byte
		if _, err := codec.MP.UnmarshalFromBytes(data, &key); err != nil {
			return err
		}
		var m getValueMessage
		if value, err := p.frame.ctx.GetValue(key); err != nil {
			p.log.Tracef("Proxy[%p].GetValue key=<%x> err=%+v", p, key, err)
			return err
		} else {
			if value != nil {
				m.Success = true
				m.Value = value
			} else {
				m.Success = false
				m.Value = nil
			}
			p.log.Tracef("Proxy[%p].GetValue key=<%x> value=<%x>", p, key, value)
		}
		return p.conn.Send(msgGETVALUE, &m)

	case msgSETVALUE:
		var m setValueMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		var old []byte
		var err error
		if (m.Flag & flagDELETE) != 0 {
			old, err = p.frame.ctx.DeleteValue(m.Key)
			p.log.Tracef("Proxy[%p].Delete key=<%x> old=<%x>", p, m.Key, old)
		} else {
			old, err = p.frame.ctx.SetValue(m.Key, m.Value)
			p.log.Tracef("Proxy[%p].SetValue key=<%x> value=<%x> old=<%x>", p, m.Key, m.Value, old)
		}
		if err != nil {
			return err
		}
		if (m.Flag & flagOLDVALUE) != 0 {
			var ret = oldValueMessage{
				HasOld:  old != nil,
				OldSize: len(old),
			}
			return p.conn.Send(msgSETVALUE, &ret)
		} else {
			return nil
		}

	case msgCALL:
		var m callMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		p.log.Tracef("Proxy[%p].OnCall from=%v to=%v value=%v limit=%v type=%s data=%v",
			p, p.frame.addr, &m.To, &m.Value.Int, &m.Limit.Int, m.DataType, m.Data)
		p.frame.ctx.OnCall(p.frame.addr, &m.To, &m.Value.Int, &m.Limit.Int, m.DataType, m.Data)
		return nil

	case msgEVENT:
		var m eventMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		p.log.Tracef("Proxy[%p].OnEvent from=%v indexed=%v data=%v",
			p, p.frame.addr, m.Indexed, m.Data)
		return p.frame.ctx.OnEvent(p.frame.addr, m.Indexed, m.Data)

	case msgGETBALANCE:
		var addr common.Address
		if _, err := codec.MP.UnmarshalFromBytes(data, &addr); err != nil {
			return err
		}
		var balance common.HexInt
		balance.Set(p.frame.ctx.GetBalance(&addr))
		p.log.Tracef("Proxy[%p].GetBalance(%s) -> %s",
			p, &addr, &balance)
		return p.conn.Send(msgGETBALANCE, &balance)

	case msgGETAPI:
		var m getAPIMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		p.log.Tracef("Proxy[%p].OnAPI status=%s, info=%s",
			p, module.Status(m.Status), m.Info)

		frame := p.popFrame()
		if frame == nil {
			return errors.InvalidStateError.New("Empty frame")
		}

		var status error
		if m.Status != errors.Success {
			status = m.Status.New(module.Status(m.Status).String())
		}
		frame.ctx.OnAPI(status, m.Info)
		return p.tryToBeReady()

	case msgLOG:
		var m logMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}

		p.lock.Lock()
		defer p.lock.Unlock()

		if (m.Flag & LogFlagTrace) != 0 {
			p.log.TLog(traceLevelOf(m.Level), m.Message)
		}
		if p.frame != nil && p.frame.addr != nil {
			p.log.Log(m.Level, p.scoreType, "|",
				common.StrLeft(10, p.frame.addr.String()), "|", m.Message)
		} else {
			p.log.Log(m.Level, p.scoreType, "|", m.Message)
		}
		return nil

	case msgSETCODE:
		var code []byte
		if _, err := codec.MP.UnmarshalFromBytes(data, &code); err != nil {
			return err
		}
		return p.frame.ctx.SetCode(code)

	case msgGETOBJGRAPH:
		var flags int
		if _, err := codec.MP.UnmarshalFromBytes(data, &flags); err != nil {
			p.log.Debugf("Failed to UnmarshalFromBytes err(%s)\n", err)
			return err
		}
		nextHash, graphHash, objGraph, err := p.frame.ctx.GetObjGraph(flags == 1)
		if err != nil {
			p.log.Debugf("Failed to getObjGraph err(%s)\n", err)
			return err
		}
		m := getObjGraphMessage{
			NextHash:    nextHash,
			GraphHash:   graphHash,
			ObjectGraph: objGraph,
		}
		return p.conn.Send(msgGETOBJGRAPH, &m)

	case msgSETOBJGRAPH:
		var m setObjGraphMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		return p.frame.ctx.SetObjGraph(m.Flags == 1, m.NextHash, m.ObjectGraph)

	case msgSETFEEPCT:
		var proportion int
		if _, err := codec.MP.UnmarshalFromBytes(data, &proportion); err != nil {
			return err
		}
		if 0 <= proportion && proportion <= 100 {
			p.frame.ctx.OnSetFeeProportion(proportion)
		} else {
			p.log.Warnf("Proxy[%p].OnSetFeeProportion: invalid proportion=%d",
				proportion)
		}
		return nil

	case msgCONTAINS:
		var m containsMessage
		if _, err := codec.MP.UnmarshalFromBytes(data, &m); err != nil {
			return err
		}
		yn, cnt, sz, err := p.frame.ctx.ArrayDBContains(m.Prefix, m.Value, m.Limit)
		if err != nil {
			p.log.Tracef("Proxy[%p].Contains prefix=<%x> value=<%x> limit=<%d> err=%+v",
				p, m.Prefix, m.Value, m.Limit, err)
			return err
		}
		res := containsResponse{
			YN:    yn,
			Count: cnt,
			Size:  sz,
		}
		p.log.Tracef("Proxy[%p].Contains prefix=<%x> value=<%x> limit=<%d> yn=<%t> cnt=<%d> sz=<%d>",
			p, m.Prefix, m.Value, m.Limit, yn, cnt, sz)
		return p.conn.Send(msgCONTAINS, &res)

	default:
		p.log.Warnf("Proxy[%p].HandleMessage(msg=%d) UnknownMessage", msg)
		return errors.ErrIllegalArgument
	}
}

func (p *proxy) setState(s proxyState) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.state = s
}

func (p *proxy) close() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.state != stateClosed {
		p.state = stateClosed
		p.conn.Send(msgCLOSE, nil)
		return p.conn.Close()
	}
	return nil
}

func (p *proxy) OnClose() {
	l := common.LockForAutoCall(&p.lock)
	defer l.Unlock()

	if p.frame != nil && p.state == stateReserved {
		frame := p.frame
		status := errors.ExecutionFailError.New("ProxyIsClosed")
		l.CallAfterUnlock(func() {
			frame.ctx.OnResult(status, 0, new(big.Int), nil)
		})
	}

	if p.state != stateClosed {
		p.state = stateClosed
	}
}

func (p *proxy) Kill() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.log.Warnf("Proxy[%p].Kill() type=%s uid=%s", p, p.scoreType, p.uid)
	p.state = stateStopped
	return p.mgr.kill(p.uid)
}

func (p *proxy) detach() bool {
	if p.pprev == nil {
		return false
	}
	*p.pprev = p.next
	if p.next != nil {
		p.next.pprev = p.pprev
	}
	p.pprev = nil
	p.next = nil
	return true
}

func (p *proxy) attachTo(r **proxy) {
	p.next = *r
	if p.next != nil {
		p.next.pprev = &p.next
	}
	p.pprev = r
	*r = p
}

func newProxy(m proxyManager, c ipc.Connection, l log.Logger, t string, v uint16, uid string) (*proxy, error) {
	logger := trace.LoggerOf(l.WithFields(log.Fields{
		log.FieldKeyEID: uid,
	}))
	p := &proxy{
		mgr:  m,
		conn: c,
		log:  logger,

		scoreType: t,
		version:   v,
		uid:       uid,
		state:     stateIdle,
	}
	c.SetHandler(msgRESULT, p)
	c.SetHandler(msgGETVALUE, p)
	c.SetHandler(msgSETVALUE, p)
	c.SetHandler(msgCALL, p)
	c.SetHandler(msgEVENT, p)
	c.SetHandler(msgGETINFO, p)
	c.SetHandler(msgGETBALANCE, p)
	c.SetHandler(msgGETAPI, p)
	c.SetHandler(msgLOG, p)
	c.SetHandler(msgSETCODE, p)
	c.SetHandler(msgGETOBJGRAPH, p)
	c.SetHandler(msgSETOBJGRAPH, p)
	c.SetHandler(msgSETFEEPCT, p)
	c.SetHandler(msgCONTAINS, p)

	if err := m.onReady(p); err != nil {
		p.state = stateStopped
		return nil, err
	}
	p.state = stateReady
	return p, nil
}

func newUID() string {
	return uuid.Must(uuid.NewV4()).String()
}
